package server

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mostlygeek/llama-swap/internal/config"
	"github.com/mostlygeek/llama-swap/internal/swaputil"
	"github.com/tidwall/gjson"
)

func TestServer_ApplyFilters(t *testing.T) {
	t.Run("useModelName rewrite", func(t *testing.T) {
		out, err := applyFilters([]byte(`{"model":"alias","temp":1}`), "alias", "real-model", config.Filters{}, config.ReasoningEffortConfig{})
		if err != nil {
			t.Fatalf("applyFilters: %v", err)
		}
		if got := gjson.GetBytes(out, "model").String(); got != "real-model" {
			t.Errorf("model = %q, want real-model", got)
		}
	})

	t.Run("strip and set params", func(t *testing.T) {
		f := config.Filters{
			StripParams: "temperature",
			SetParams:   map[string]any{"top_p": 0.9},
		}
		out, err := applyFilters([]byte(`{"model":"m","temperature":0.7}`), "m", "", f, config.ReasoningEffortConfig{})
		if err != nil {
			t.Fatalf("applyFilters: %v", err)
		}
		if gjson.GetBytes(out, "temperature").Exists() {
			t.Error("temperature should be stripped")
		}
		if got := gjson.GetBytes(out, "top_p").Float(); got != 0.9 {
			t.Errorf("top_p = %v, want 0.9", got)
		}
	})

	t.Run("setParamsByID overrides setParams", func(t *testing.T) {
		f := config.Filters{
			SetParams:     map[string]any{"top_p": 0.5},
			SetParamsByID: map[string]map[string]any{"alias": {"top_p": 0.1}},
		}
		out, err := applyFilters([]byte(`{"model":"alias"}`), "alias", "", f, config.ReasoningEffortConfig{})
		if err != nil {
			t.Fatalf("applyFilters: %v", err)
		}
		if got := gjson.GetBytes(out, "top_p").Float(); got != 0.1 {
			t.Errorf("top_p = %v, want 0.1", got)
		}
	})
}

func TestServer_ApplyReasoningEffort(t *testing.T) {
	enabled := config.ReasoningEffortConfig{
		Enable:  true,
		Budgets: config.DefaultReasoningEffortBudgets(),
	}
	disabled := config.ReasoningEffortConfig{}

	tests := []struct {
		name             string
		body             string
		cfg              config.ReasoningEffortConfig
		wantBudgetExists bool
		wantBudget       int
		wantEffortKept   bool
	}{
		{name: "off -> 0", body: `{"model":"m","reasoning_effort":"off"}`, cfg: enabled, wantBudgetExists: true, wantBudget: 0, wantEffortKept: false},
		{name: "low -> 512", body: `{"model":"m","reasoning_effort":"low"}`, cfg: enabled, wantBudgetExists: true, wantBudget: 512, wantEffortKept: false},
		{name: "medium -> 2048", body: `{"model":"m","reasoning_effort":"medium"}`, cfg: enabled, wantBudgetExists: true, wantBudget: 2048, wantEffortKept: false},
		{name: "high -> 8192", body: `{"model":"m","reasoning_effort":"high"}`, cfg: enabled, wantBudgetExists: true, wantBudget: 8192, wantEffortKept: false},
		{name: "max -> -1", body: `{"model":"m","reasoning_effort":"max"}`, cfg: enabled, wantBudgetExists: true, wantBudget: -1, wantEffortKept: false},
		{name: "missing field -> untouched", body: `{"model":"m"}`, cfg: enabled, wantBudgetExists: false, wantEffortKept: false},
		{name: "unknown value -> untouched", body: `{"model":"m","reasoning_effort":"minimal"}`, cfg: enabled, wantBudgetExists: false, wantEffortKept: true},
		{name: "disabled -> untouched", body: `{"model":"m","reasoning_effort":"low"}`, cfg: disabled, wantBudgetExists: false, wantEffortKept: true},
		{name: "case-insensitive", body: `{"model":"m","reasoning_effort":"LOW"}`, cfg: enabled, wantBudgetExists: true, wantBudget: 512, wantEffortKept: false},
		{name: "non-string value -> untouched", body: `{"model":"m","reasoning_effort":5}`, cfg: enabled, wantBudgetExists: false, wantEffortKept: true},
		{name: "explicit budget wins over effort", body: `{"model":"m","reasoning_budget_tokens":123,"reasoning_effort":"low"}`, cfg: enabled, wantBudgetExists: true, wantBudget: 123, wantEffortKept: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := applyReasoningEffort([]byte(tt.body), tt.cfg)
			if err != nil {
				t.Fatalf("applyReasoningEffort: %v", err)
			}

			budget := gjson.GetBytes(out, "reasoning_budget_tokens")
			if tt.wantBudgetExists {
				if !budget.Exists() || budget.Int() != int64(tt.wantBudget) {
					t.Errorf("reasoning_budget_tokens = %s, want %d", budget.Raw, tt.wantBudget)
				}
			} else if budget.Exists() {
				t.Errorf("reasoning_budget_tokens = %s, want absent", budget.Raw)
			}

			effort := gjson.GetBytes(out, "reasoning_effort")
			if tt.wantEffortKept && !effort.Exists() {
				t.Errorf("reasoning_effort was removed, want kept")
			}
			if !tt.wantEffortKept && effort.Exists() {
				t.Errorf("reasoning_effort = %s, want removed", effort.Raw)
			}

			// model must survive the rewrite in every case
			if got := gjson.GetBytes(out, "model").String(); got != "m" {
				t.Errorf("model = %q, want m", got)
			}
		})
	}
}

func TestReasoningEffortConfig_BudgetFor(t *testing.T) {
	cfg := config.ReasoningEffortConfig{Budgets: config.DefaultReasoningEffortBudgets()}

	tests := []struct {
		effort string
		want   int
		ok     bool
	}{
		{"off", 0, true},
		{"low", 512, true},
		{"medium", 2048, true},
		{"high", 8192, true},
		{"max", -1, true},
		{"High", 8192, true}, // case-insensitive
		{" low ", 512, true}, // trims whitespace
		{"", 0, false},
		{"minimal", 0, false},
		{"xhigh", 0, false},
	}

	for _, tt := range tests {
		budget, ok := cfg.BudgetFor(tt.effort)
		if ok != tt.ok || budget != tt.want {
			t.Errorf("BudgetFor(%q) = (%d, %v), want (%d, %v)", tt.effort, budget, ok, tt.want, tt.ok)
		}
	}

	// custom mapping overrides defaults for the same key, and falls back to
	// defaults for keys it does not define (merge semantics)
	custom := config.ReasoningEffortConfig{Budgets: map[string]int{"low": 100}}
	if budget, ok := custom.BudgetFor("low"); !ok || budget != 100 {
		t.Errorf("custom BudgetFor(low) = (%d, %v), want (100, true)", budget, ok)
	}
	if budget, ok := custom.BudgetFor("high"); !ok || budget != 8192 {
		t.Errorf("custom BudgetFor(high) = (%d, %v), want (8192, true) from default", budget, ok)
	}
	if _, ok := custom.BudgetFor("minimal"); ok {
		t.Error("custom BudgetFor(minimal) should not exist (not in defaults either)")
	}
}

func TestServer_ResolveFilters_QualifiedPeer(t *testing.T) {
	want := config.Filters{StripParams: "temperature"}
	cfg := config.Config{Peers: config.PeerDictionaryConfig{
		"remote": {
			Models:  []string{"org/model"},
			Filters: want,
		},
	}}

	useModelName, got, ok := resolveFilters(cfg, "remote/org/model")
	if !ok {
		t.Fatal("qualified peer filters were not resolved")
	}
	if useModelName != "" {
		t.Fatalf("useModelName = %q, want empty for peer", useModelName)
	}
	if got.StripParams != want.StripParams {
		t.Fatalf("StripParams = %q, want %q", got.StripParams, want.StripParams)
	}
}

func TestServer_FormFilterMiddleware(t *testing.T) {
	cfg := config.Config{Models: map[string]config.ModelConfig{
		"whisper": {UseModelName: "whisper-large-v3"},
	}}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("model", "whisper")
	fw, _ := mw.CreateFormFile("file", "a.wav")
	fw.Write([]byte("xx"))
	mw.Close()

	r := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())

	var gotModel, gotFilename, gotFileBody string
	var gotContext swaputil.ReqContextData
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(swaputil.MaxMultiPartSize); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			return
		}
		gotModel = r.MultipartForm.Value["model"][0]
		fileHeader := r.MultipartForm.File["file"][0]
		gotFilename = fileHeader.Filename
		file, err := fileHeader.Open()
		if err != nil {
			t.Errorf("open file: %v", err)
			return
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			t.Errorf("read file: %v", err)
			return
		}
		gotFileBody = string(data)
		gotContext, _ = swaputil.ReadContext(r.Context())
	})
	CreateFormFilterMiddleware(cfg)(final).ServeHTTP(httptest.NewRecorder(), r)

	if gotModel != "whisper-large-v3" {
		t.Errorf("model rewritten to %q, want whisper-large-v3", gotModel)
	}
	if gotFilename != "a.wav" {
		t.Errorf("filename = %q, want a.wav", gotFilename)
	}
	if gotFileBody != "xx" {
		t.Errorf("file body = %q, want xx", gotFileBody)
	}
	if gotContext.Model != "whisper" || gotContext.ModelID != "whisper" {
		t.Errorf("request context = %+v, want original whisper model", gotContext)
	}
}
