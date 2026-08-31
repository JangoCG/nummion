package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"lexware-cli/internal/api"
)

type pageEnvelope struct {
	Content          []map[string]any `json:"content"`
	First            bool             `json:"first"`
	Last             bool             `json:"last"`
	TotalPages       int              `json:"totalPages"`
	TotalElements    int              `json:"totalElements"`
	NumberOfElements int              `json:"numberOfElements"`
	Size             int              `json:"size"`
	Number           int              `json:"number"`
	Sort             any              `json:"sort,omitempty"`
}

func validatePaging(page, size int) error {
	if page < 0 {
		return fmt.Errorf("--page darf nicht negativ sein")
	}
	if size < 1 || size > 250 {
		return fmt.Errorf("--size muss zwischen 1 und 250 liegen")
	}
	return nil
}

func fetchPage(ctx context.Context, client *api.Client, path string, query url.Values, all bool) (json.RawMessage, pageEnvelope, error) {
	requestQuery := cloneValues(query)
	if !all {
		raw, err := client.JSON(ctx, http.MethodGet, path, requestQuery, nil)
		if err != nil {
			return nil, pageEnvelope{}, err
		}
		var page pageEnvelope
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, pageEnvelope{}, err
		}
		return raw, page, nil
	}

	requestQuery.Set("page", "0")
	combined := pageEnvelope{First: true, Last: true, Number: 0}
	for pageNumber := 0; ; pageNumber++ {
		requestQuery.Set("page", strconv.Itoa(pageNumber))
		raw, err := client.JSON(ctx, http.MethodGet, path, requestQuery, nil)
		if err != nil {
			return nil, pageEnvelope{}, err
		}
		var page pageEnvelope
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, pageEnvelope{}, err
		}
		combined.Content = append(combined.Content, page.Content...)
		combined.TotalPages = page.TotalPages
		combined.TotalElements = page.TotalElements
		combined.Sort = page.Sort
		if page.Last || pageNumber+1 >= page.TotalPages {
			break
		}
	}
	combined.NumberOfElements = len(combined.Content)
	combined.Size = len(combined.Content)
	raw, err := json.Marshal(combined)
	return raw, combined, err
}

func cloneValues(source url.Values) url.Values {
	cloned := make(url.Values, len(source))
	for key, values := range source {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}
