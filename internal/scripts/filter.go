package scripts

import (
	"fmt"
	"strings"
)

// batchFilterGroups maps a group alias to checker script IDs.
var batchFilterGroups = map[string][]string{
	"email":    {"chk-smtp", "chk-sendgrid", "chk-mailgun"},
	"mail":     {"chk-smtp", "chk-sendgrid", "chk-mailgun"},
	"db":       {"chk-mysql", "chk-postgres", "chk-mongodb"},
	"database": {"chk-mysql", "chk-postgres", "chk-mongodb"},
	"cache":    {"chk-redis", "chk-memcached"},
	"llm":      {"chk-openai", "chk-gemini", "chk-groq", "chk-claud", "chk-xai", "chk-mistral", "chk-deepseek", "chk-perplexity", "chk-openrouter", "chk-together", "chk-cohere", "chk-replicate", "chk-huggingface"},
	"pay":      {"chk-stripe", "chk-paypal", "chk-razorpay", "chk-paystack", "chk-paddle", "chk-flutterwave"},
	"cloud":    {"chk-aws", "chk-firebase", "chk-supabase", "chk-sentry"},
	"msg":      {"chk-twilio", "chk-nexmo", "chk-telegram", "chk-fcm", "chk-pusher"},
	"crm":      {"chk-hubspot", "chk-shopify"},
}

// batchFilterAliases maps a short name to one script ID.
var batchFilterAliases = map[string]string{
	"mysql":      "chk-mysql",
	"postgres":   "chk-postgres",
	"postgresql": "chk-postgres",
	"pg":         "chk-postgres",
	"mongodb":    "chk-mongodb",
	"mongo":      "chk-mongodb",
	"redis":      "chk-redis",
	"memcached":  "chk-memcached",
	"smtp":       "chk-smtp",
	"sendgrid":   "chk-sendgrid",
	"mailgun":    "chk-mailgun",
	"twilio":     "chk-twilio",
	"stripe":     "chk-stripe",
	"aws":        "chk-aws",
	"openai":     "chk-openai",
	"gemini":     "chk-gemini",
}

// ResolveBatchFilter turns --filter values into script IDs.
// Accepts group names (db, email), short names (mysql, smtp) or full IDs (chk-mysql).
func ResolveBatchFilter(raw string) ([]string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return nil, fmt.Errorf("filter vazio")
	}
	if ids, ok := batchFilterGroups[key]; ok {
		return append([]string(nil), ids...), nil
	}
	if id, ok := batchFilterAliases[key]; ok {
		return []string{id}, nil
	}
	if strings.HasPrefix(key, "chk-") {
		return []string{key}, nil
	}
	return nil, fmt.Errorf("filter desconhecido %q — exemplos: mysql, db, email, chk-smtp", raw)
}

func scriptAllowed(id string, opts BatchPlanOpts) bool {
	if opts.ScriptID != "" {
		return id == opts.ScriptID
	}
	if len(opts.ScriptIDs) == 0 {
		return true
	}
	for _, want := range opts.ScriptIDs {
		if id == want {
			return true
		}
	}
	return false
}
