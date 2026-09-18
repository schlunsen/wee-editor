package agents

import (
	"reflect"
	"testing"
)

func askUserQuestionInput() map[string]interface{} {
	return map[string]interface{}{
		"questions": []interface{}{
			map[string]interface{}{
				"question":    "Which sources?",
				"header":      "Sources",
				"multiSelect": true,
				"options": []interface{}{
					map[string]interface{}{"label": "Local HTML", "description": "Folders"},
					map[string]interface{}{"label": "Remote URLs", "description": "Live pages"},
				},
			},
			map[string]interface{}{
				"question": "Where should the app live?",
				"header":   "Project",
				"options": []interface{}{
					map[string]interface{}{"label": "Fresh project", "description": "Clean"},
					map[string]interface{}{"label": "Reuse", "description": "Existing"},
				},
			},
		},
	}
}

func TestParseAskUserQuestions_AllQuestions(t *testing.T) {
	qs, err := parseAskUserQuestions(askUserQuestionInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(qs))
	}
	if qs[0].Question != "Which sources?" || qs[0].Header != "Sources" || !qs[0].MultiSelect {
		t.Errorf("first question parsed incorrectly: %+v", qs[0])
	}
	if len(qs[0].Options) != 2 || qs[0].Options[1].Label != "Remote URLs" {
		t.Errorf("first question options parsed incorrectly: %+v", qs[0].Options)
	}
	if qs[1].Question != "Where should the app live?" || qs[1].MultiSelect {
		t.Errorf("second question parsed incorrectly: %+v", qs[1])
	}
}

func TestParseAskUserQuestions_Invalid(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"missing questions":   {},
		"empty questions":     {"questions": []interface{}{}},
		"question not object": {"questions": []interface{}{"nope"}},
		"no question text": {"questions": []interface{}{
			map[string]interface{}{"options": []interface{}{map[string]interface{}{"label": "a"}}},
		}},
		"no options": {"questions": []interface{}{
			map[string]interface{}{"question": "q?"},
		}},
	}
	for name, input := range cases {
		if _, err := parseAskUserQuestions(input); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestBuildAskUserQuestionUpdatedInput_PreservesSchema(t *testing.T) {
	original := askUserQuestionInput()
	answers := map[string]interface{}{
		"Which sources?":             formatAskUserQuestionAnswer([]string{"Local HTML", "Remote URLs"}),
		"Where should the app live?": formatAskUserQuestionAnswer([]string{"Fresh project"}),
	}

	updated := buildAskUserQuestionUpdatedInput(original, answers)

	// The required `questions` array must survive untouched.
	if !reflect.DeepEqual(updated["questions"], original["questions"]) {
		t.Errorf("questions were not preserved in updated input")
	}
	// `answers` must be an object keyed by question text with string values.
	got, ok := updated["answers"].(map[string]interface{})
	if !ok {
		t.Fatalf("answers should be a map, got %T", updated["answers"])
	}
	if got["Which sources?"] != "Local HTML, Remote URLs" {
		t.Errorf("multi-select answer not joined: %v", got["Which sources?"])
	}
	if got["Where should the app live?"] != "Fresh project" {
		t.Errorf("single answer wrong: %v", got["Where should the app live?"])
	}
	// The original input must not be mutated.
	if _, present := original["answers"]; present {
		t.Errorf("original input was mutated")
	}
}
