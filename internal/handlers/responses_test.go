package handlers_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"org.banana.project/api/internal/handlers"
	"org.banana.project/api/internal/service"
)

func TestResponseDTOs_JSONShape(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "message response includes id",
			value: handlers.MessageResponse{Status: "success", Message: "created", ID: 7},
			want:  `{"status":"success","message":"created","id":7}`,
		},
		{
			name:  "message response omits empty id",
			value: handlers.MessageResponse{Status: "success", Message: "updated"},
			want:  `{"status":"success","message":"updated"}`,
		},
		{
			name:  "sale response exposes saleId",
			value: handlers.SaleResponse{Status: "success", Message: "done", SaleID: 9},
			want:  `{"status":"success","message":"done","saleId":9}`,
		},
		{
			name: "sync response wraps result",
			value: handlers.SyncResponse{
				Status:  "success",
				Message: "done",
				Result:  service.SyncResult{Fetched: 3, Imported: 2, Skipped: 1, Errors: 0},
			},
			want: `{"status":"success","message":"done","result":{"fetched":3,"imported":2,"skipped":1,"errors":0}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("failed to marshal value: %v", err)
			}

			var gotMap, wantMap map[string]any
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("failed to unmarshal got JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantMap); err != nil {
				t.Fatalf("failed to unmarshal want JSON: %v", err)
			}

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("unexpected JSON shape\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}
