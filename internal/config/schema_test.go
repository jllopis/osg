package config

import (
	"testing"
)

func TestConfigSchema(t *testing.T) {
	sections := ConfigSchema()
	if len(sections) == 0 {
		t.Fatal("ConfigSchema() returned no sections")
	}

	t.Run("all sections have names", func(t *testing.T) {
		for i, s := range sections {
			if s.Name == "" {
				t.Errorf("section[%d] has empty Name", i)
			}
			if s.Description == "" {
				t.Errorf("section[%d] %q has empty Description", i, s.Name)
			}
		}
	})

	t.Run("all fields have keys and labels", func(t *testing.T) {
		for _, s := range sections {
			for j, f := range s.Fields {
				if f.Key == "" {
					t.Errorf("section %q field[%d] has empty Key", s.Name, j)
				}
				if f.Label == "" {
					t.Errorf("section %q field[%d] %q has empty Label", s.Name, j, f.Key)
				}
				if f.Description == "" {
					t.Errorf("section %q field[%d] %q has empty Description", s.Name, j, f.Key)
				}
			}
		}
	})

	t.Run("no duplicate keys", func(t *testing.T) {
		seen := map[string]bool{}
		for _, s := range sections {
			for _, f := range s.Fields {
				if seen[f.Key] {
					t.Errorf("duplicate key: %q", f.Key)
				}
				seen[f.Key] = true
			}
		}
	})

	t.Run("struct list fields have nested", func(t *testing.T) {
		for _, s := range sections {
			for _, f := range s.Fields {
				if f.Type == FieldStructList && len(f.Nested) == 0 {
					t.Errorf("section %q field %q is FieldStructList but has no Nested fields", s.Name, f.Key)
				}
			}
		}
	})

	t.Run("covers key config fields", func(t *testing.T) {
		// Ensure critical fields are present in the schema.
		required := []string{
			"site_title", "base_url", "theme", "vault_path",
			"content_dir", "public_dir", "ai.provider",
			"logging.level", "interactions.enabled",
			"interactions.comments.enabled", "plugins_enabled",
		}
		for _, key := range required {
			if _, ok := FindField(key); !ok {
				t.Errorf("required field %q not found in schema", key)
			}
		}
	})
}

func TestAllFields(t *testing.T) {
	fields := AllFields()
	if len(fields) == 0 {
		t.Fatal("AllFields() returned no fields")
	}
	// Should be the sum of all section field counts.
	total := 0
	for _, s := range ConfigSchema() {
		total += len(s.Fields)
	}
	if len(fields) != total {
		t.Errorf("AllFields() = %d fields; want %d", len(fields), total)
	}
}

func TestFindField(t *testing.T) {
	t.Run("existing field", func(t *testing.T) {
		f, ok := FindField("site_title")
		if !ok {
			t.Fatal("FindField(\"site_title\") returned false")
		}
		if f.Label != "Site Title" {
			t.Errorf("Label = %q; want \"Site Title\"", f.Label)
		}
	})

	t.Run("nested field", func(t *testing.T) {
		f, ok := FindField("ai.provider")
		if !ok {
			t.Fatal("FindField(\"ai.provider\") returned false")
		}
		if f.Type != FieldString {
			t.Errorf("Type = %d; want FieldString", f.Type)
		}
		if len(f.Options) == 0 {
			t.Error("ai.provider should have Options")
		}
	})

	t.Run("nonexistent field", func(t *testing.T) {
		_, ok := FindField("nonexistent.field")
		if ok {
			t.Error("FindField(\"nonexistent.field\") should return false")
		}
	})

	t.Run("sensitive field", func(t *testing.T) {
		f, ok := FindField("ai.api_key")
		if !ok {
			t.Fatal("FindField(\"ai.api_key\") returned false")
		}
		if !f.Sensitive {
			t.Error("ai.api_key should be marked Sensitive")
		}
	})
}
