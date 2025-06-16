package test

import "encoding/json"

// KeyWithVariants connects the core unique db testID string to a set of variants.
// Used to serialize/deserialize as a map key when we pass test status around.
type KeyWithVariants struct {
	TestID string `json:"test_id"`

	// Proposed, need to serialize to use as map key
	Variants map[string]string `json:"variants"`
}

// KeyOrDie serializes this test key into a json string suitable for use in maps.
// JSON serialization uses sorted map keys, so the output is stable.
func (t KeyWithVariants) KeyOrDie() string {
	testIDBytes, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return string(testIDBytes)
}
