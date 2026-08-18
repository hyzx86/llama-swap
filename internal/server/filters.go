package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mostlygeek/llama-swap/internal/chain"
	"github.com/mostlygeek/llama-swap/internal/config"
	"github.com/mostlygeek/llama-swap/internal/swaputil"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// CreateFilterMiddleware returns middleware that applies per-model request-body
// filters to JSON requests before they are forwarded upstream:
//
//   - UseModelName rewrite (issue #69)
//   - StripParams removal (issue #174)
//   - SetParams injection (issue #453)
//   - SetParamsByID per-alias overrides
//
// Non-JSON requests (GET, multipart forms) pass through untouched. The buffered
// body is re-attached with Content-Length / Transfer-Encoding cleanup so the
// downstream reverse proxy forwards the correct bytes (see issue #11).
func CreateFilterMiddleware(cfg config.Config) chain.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
				next.ServeHTTP(w, r)
				return
			}

			data, err := swaputil.FetchContext(r, cfg)
			if err != nil {
				swaputil.SendError(w, r, swaputil.ErrNoModelInContext)
				return
			}

			useModelName, filters, ok := resolveFilters(cfg, data.Model)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				swaputil.SendResponse(w, r, http.StatusBadRequest, "could not read request body")
				return
			}

			body, err = applyFilters(body, data.Model, useModelName, filters, cfg.ReasoningEffort)
			if err != nil {
				swaputil.SendResponse(w, r, http.StatusInternalServerError, err.Error())
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			r.Header.Del("Transfer-Encoding")
			r.Header.Set("Content-Length", strconv.Itoa(len(body)))
			r.ContentLength = int64(len(body))

			next.ServeHTTP(w, r)
		})
	}
}

// CreateFormFilterMiddleware returns middleware that applies the UseModelName
// rewrite (issue #69) to multipart/form-data requests before they are forwarded
// upstream. JSON-body filters (StripParams, SetParams) do not apply to form
// endpoints; only the "model" field is rewritten.
//
// Non-multipart requests pass through untouched. When a rewrite is needed the
// form is reconstructed and re-attached with Content-Type / Content-Length
// cleanup so the downstream reverse proxy forwards the correct bytes.
func CreateFormFilterMiddleware(cfg config.Config) chain.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				next.ServeHTTP(w, r)
				return
			}

			data, err := swaputil.FetchContext(r, cfg)
			if err != nil {
				swaputil.SendError(w, r, swaputil.ErrNoModelInContext)
				return
			}

			useModelName, _, ok := resolveFilters(cfg, data.Model)
			if !ok || useModelName == "" {
				next.ServeHTTP(w, r)
				return
			}

			updated, err := swaputil.ReplaceRequestModel(r, data.Model, useModelName)
			if err != nil {
				swaputil.SendResponse(w, r, http.StatusBadRequest, err.Error())
				return
			}

			// UseModelName changes only the model name sent upstream. Keep the
			// original request context so routing and metrics still identify
			// the configured model.
			updated = updated.WithContext(r.Context())
			next.ServeHTTP(w, updated)
		})
	}
}

// resolveFilters returns the filter settings for a requested model. UseModelName
// only applies to local models; peers carry filters but no name rewrite.
func resolveFilters(cfg config.Config, requested string) (useModelName string, filters config.Filters, ok bool) {
	if realName, found := cfg.RealModelName(requested); found {
		mc := cfg.Models[realName]
		return mc.UseModelName, mc.Filters.Filters, true
	}
	if peerID, _, found := cfg.ResolvePeerModel(requested); found {
		return "", cfg.Peers[peerID].Filters, true
	}
	return "", config.Filters{}, false
}

// applyFilters rewrites the JSON body in place. Order matches the legacy
// ProxyManager: useModelName, stripParams, setParams, then setParamsByID (which
// can override setParams). The reasoning-effort translation (if enabled) runs
// last so it takes precedence over any setParams override of the budget field.
func applyFilters(body []byte, requested, useModelName string, f config.Filters, reasoningEffort config.ReasoningEffortConfig) ([]byte, error) {
	var err error

	if useModelName != "" {
		if body, err = sjson.SetBytes(body, "model", useModelName); err != nil {
			return nil, fmt.Errorf("error rewriting model name in JSON: %w", err)
		}
	}

	for _, param := range f.SanitizedStripParams() {
		if body, err = sjson.DeleteBytes(body, param); err != nil {
			return nil, fmt.Errorf("error stripping parameter %s from request", param)
		}
	}

	setParams, setKeys := f.SanitizedSetParams()
	for _, key := range setKeys {
		if body, err = sjson.SetBytes(body, key, setParams[key]); err != nil {
			return nil, fmt.Errorf("error setting parameter %s in request", key)
		}
	}

	byID, byIDKeys := f.SanitizedSetParamsByID(requested)
	for _, key := range byIDKeys {
		if body, err = sjson.SetBytes(body, key, byID[key]); err != nil {
			return nil, fmt.Errorf("error setting parameter %s in request", key)
		}
	}

	return applyReasoningEffort(body, reasoningEffort)
}

// applyReasoningEffort translates a top-level OpenAI-style "reasoning_effort"
// field (sent by clients such as VS Code) into the llama.cpp request-level
// "reasoning_budget_tokens" field, then removes the original field so the
// incompatible value is not forwarded upstream.
//
// Behaviour:
//   - When the translation is disabled, the body is returned unchanged.
//   - If the request already carries an explicit reasoning_budget_tokens, it is
//     used as-is and no conversion happens (the client's value wins).
//   - Otherwise, if the request carries reasoning_effort, a known effort maps to
//     its configured budget, which is written to reasoning_budget_tokens
//     (llama.cpp also accepts the alias thinking_budget_tokens) and the original
//     reasoning_effort field is removed. 0 disables reasoning, -1 means
//     unlimited.
//   - An unknown effort (e.g. "minimal", "xhigh") is left untouched: the field
//     is preserved and no budget is set, so behaviour is unchanged for
//     non-reasoning models.
//   - A request with neither field is returned unchanged.
//
// The original reasoning_effort semantics are not altered; this only rewrites
// the wire format for llama.cpp.
func applyReasoningEffort(body []byte, cfg config.ReasoningEffortConfig) ([]byte, error) {
	if !cfg.Enable {
		return body, nil
	}

	// An explicit reasoning_budget_tokens from the client always wins; do not
	// convert or strip anything.
	if gjson.GetBytes(body, "reasoning_budget_tokens").Exists() {
		return body, nil
	}

	effort := gjson.GetBytes(body, "reasoning_effort")
	if !effort.Exists() {
		return body, nil
	}

	budget, ok := cfg.BudgetFor(effort.String())
	if !ok {
		// Unknown value: leave the request untouched (keep reasoning_effort,
		// do not set a budget).
		return body, nil
	}

	var err error
	if body, err = sjson.SetBytes(body, "reasoning_budget_tokens", budget); err != nil {
		return nil, fmt.Errorf("error setting reasoning_budget_tokens in request: %w", err)
	}
	if body, err = sjson.DeleteBytes(body, "reasoning_effort"); err != nil {
		return nil, fmt.Errorf("error stripping reasoning_effort from request: %w", err)
	}
	return body, nil
}
