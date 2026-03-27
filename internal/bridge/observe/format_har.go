package observe

import (
	"encoding/json"
	"fmt"
	"io"
)

func init() {
	RegisterFormat("har", func(creatorName, creatorVersion string) ExportEncoder {
		return &harEncoder{
			creator: harCreator{Name: creatorName, Version: creatorVersion},
		}
	})
}

// harEncoder writes HAR 1.2 JSON incrementally.
type harEncoder struct {
	creator    harCreator
	w          io.Writer
	entryCount int
}

type harCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (e *harEncoder) ContentType() string    { return "application/har+json" }
func (e *harEncoder) FileExtension() string  { return ".har" }

func (e *harEncoder) Start(w io.Writer) error {
	e.w = w
	header := fmt.Sprintf(
		`{"log":{"version":"1.2","creator":%s,"entries":[`,
		mustMarshal(e.creator),
	)
	_, err := io.WriteString(e.w, header)
	return err
}

func (e *harEncoder) Encode(entry ExportEntry) error {
	if e.entryCount > 0 {
		if _, err := io.WriteString(e.w, ","); err != nil {
			return err
		}
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = e.w.Write(data)
	e.entryCount++
	return err
}

func (e *harEncoder) Finish() error {
	_, err := io.WriteString(e.w, "]}}\n")
	return err
}

func mustMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
