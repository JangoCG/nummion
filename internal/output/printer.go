package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

type Printer struct {
	Out   io.Writer
	JSON  bool
	Quiet bool
}

func (p Printer) Data(raw json.RawMessage) error {
	if p.JSON || p.Quiet {
		_, err := fmt.Fprintln(p.Out, string(raw))
		return err
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(p.Out, string(encoded))
	return err
}

func (p Printer) Object(value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return p.Data(encoded)
}

func (p Printer) Success(message string) error {
	if p.JSON {
		return p.Object(map[string]any{"ok": true, "message": message})
	}
	if p.Quiet {
		return nil
	}
	_, err := fmt.Fprintln(p.Out, message)
	return err
}

func (p Printer) Table(headers []string, rows [][]string) error {
	if p.JSON || p.Quiet {
		return nil
	}
	w := tabwriter.NewWriter(p.Out, 0, 4, 2, ' ', 0)
	for i, header := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, header)
	}
	fmt.Fprintln(w)
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, cell)
		}
		fmt.Fprintln(w)
	}
	return w.Flush()
}
