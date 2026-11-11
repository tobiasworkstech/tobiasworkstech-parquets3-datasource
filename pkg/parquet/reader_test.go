package parquet

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	parquetformat "github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/compress"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestReadParquetFromS3WithListColumn(t *testing.T) {
	payload := listParquetData(t)
	client := s3.New(s3.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("https://s3.example.test"),
		UsePathStyle: true,
		Credentials:  aws.AnonymousCredentials{},
		HTTPClient: &http.Client{Transport: parquetTestTransport(func(request *http.Request) (*http.Response, error) {
			// ServeContent implements the HEAD and ranged GET requests used by the reader.
			response := httptest.NewRecorder()
			http.ServeContent(response, request, "lists.parquet", time.Time{}, bytes.NewReader(payload))
			return response.Result(), nil
		})},
	})

	t.Run("schema", func(t *testing.T) {
		frame, err := ReadParquetSchemaFromS3(context.Background(), client, "test-bucket", "lists.parquet")
		if err != nil {
			t.Fatal(err)
		}
		if len(frame.Fields) != 2 || frame.Fields[0].Name != "id" || frame.Fields[1].Name != "tags" {
			t.Fatalf("unexpected schema: %v", frame.Fields)
		}
		for _, field := range frame.Fields {
			if field.Len() != 0 {
				t.Fatalf("schema field %q contains %d rows", field.Name, field.Len())
			}
		}
	})

	t.Run("rows", func(t *testing.T) {
		frames, err := ReadParquetFromS3(context.Background(), client, "test-bucket", "lists.parquet")
		if err != nil {
			t.Fatal(err)
		}
		if len(frames) != 1 || len(frames[0].Fields) != 2 {
			t.Fatalf("unexpected frames: %v", frames)
		}
		frame := frames[0]
		if rows, err := frame.RowLen(); err != nil || rows != 3 {
			t.Fatalf("expected 3 rows, got %d: %v", rows, err)
		}
		for i, want := range []string{"first", "second", "third"} {
			got, ok := frame.Fields[0].At(i).(*string)
			if !ok || got == nil || *got != want {
				t.Errorf("id at row %d: got %v, want %q", i, frame.Fields[0].At(i), want)
			}
		}
		for i, want := range []string{`["alpha","beta"]`, "[]", array.NullValueStr} {
			if got := frame.Fields[1].At(i); got != want {
				t.Errorf("tags at row %d: got %v, want %q", i, got, want)
			}
		}
	})
}

func listParquetData(t *testing.T) []byte {
	t.Helper()
	allocator := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer allocator.AssertSize(t, 0)
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.BinaryTypes.String},
		{Name: "tags", Type: arrow.ListOf(arrow.BinaryTypes.String), Nullable: true},
	}, nil)
	builder := array.NewRecordBuilder(allocator, schema)
	defer builder.Release()
	builder.Field(0).(*array.StringBuilder).AppendValues([]string{"first", "second", "third"}, nil)
	lists := builder.Field(1).(*array.ListBuilder)
	values := lists.ValueBuilder().(*array.StringBuilder)
	lists.Append(true)
	values.AppendValues([]string{"alpha", "beta"}, nil)
	lists.Append(true)
	lists.AppendNull()
	record := builder.NewRecordBatch()
	defer record.Release()
	table := array.NewTableFromRecords(schema, []arrow.RecordBatch{record})
	defer table.Release()
	var output bytes.Buffer
	properties := parquetformat.NewWriterProperties(parquetformat.WithCompression(compress.Codecs.Zstd))
	if err := pqarrow.WriteTable(table, &output, 2, properties, pqarrow.DefaultWriterProps()); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

type parquetTestTransport func(*http.Request) (*http.Response, error)

func (transport parquetTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}
