package client

import (
	"encoding/json"
	"fmt"

	"github.com/OpenSLX/bwlp-go-client/bwlp"
)

type BwlpSpecifics struct {
	Details *bwlp.ImageDetailsRead
	Version *bwlp.ImageVersionDetails
}

func (spec *BwlpSpecifics) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&spec.Details, &spec.Version}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf("Wrong number of fields for %T: Expected %d, got %d.\n", spec, g, e)
	}
	return nil
}
