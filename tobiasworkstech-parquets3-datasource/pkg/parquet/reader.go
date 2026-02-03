package parquet

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type S3ReaderAt struct {
	Ctx      context.Context
	S3Client *s3.Client
	Bucket   string
	Key      string
	Size     int64
	offset   int64
}

func (r *S3ReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= r.Size {
		return 0, io.EOF
	}
	end := off + int64(len(p)) - 1
	if end >= r.Size {
		end = r.Size - 1
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", off, end)
	res, err := r.S3Client.GetObject(r.Ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.Bucket),
		Key:    aws.String(r.Key),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	return io.ReadFull(res.Body, p)
}

func (r *S3ReaderAt) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = r.Size + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}
	return r.offset, nil
}

func ReadParquetFromS3(ctx context.Context, s3Client *s3.Client, bucket, key string) ([]*data.Frame, error) {
	head, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("head object: %w", err)
	}

	readerAt := &S3ReaderAt{
		Ctx:      ctx,
		S3Client: s3Client,
		Bucket:   bucket,
		Key:      key,
		Size:     *head.ContentLength,
	}

	pf, err := file.NewParquetReader(readerAt)
	if err != nil {
		return nil, fmt.Errorf("new parquet reader: %w", err)
	}

	arrowReader, err := pqarrow.NewFileReader(pf, pqarrow.ArrowReadProperties{}, nil)
	if err != nil {
		return nil, fmt.Errorf("new arrow reader: %w", err)
	}

	table, err := arrowReader.ReadTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("read table: %w", err)
	}
	defer table.Release()

	// Convert Arrow Table to Grafana Data Frame
	// We iterate over columns and create fields with proper types.

	frame := data.NewFrame(key)
	numRows := int(table.NumRows())

	for i := 0; i < int(table.NumCols()); i++ {
		col := table.Column(i)
		fieldName := table.Schema().Field(i).Name
		fieldType := table.Schema().Field(i).Type

		field := convertArrowColumnToField(fieldName, col, fieldType, numRows)
		frame.Fields = append(frame.Fields, field)
	}

	return []*data.Frame{frame}, nil
}

// convertArrowColumnToField converts an Arrow column to a Grafana data field with proper typing
func convertArrowColumnToField(name string, col *arrow.Column, fieldType arrow.DataType, numRows int) *data.Field {
	// Handle chunked arrays by combining all chunks
	chunks := col.Data().Chunks()

	switch fieldType.ID() {
	case arrow.TIMESTAMP:
		values := make([]*time.Time, numRows)
		idx := 0
		for _, chunk := range chunks {
			tsArray := chunk.(*array.Timestamp)
			tsType := fieldType.(*arrow.TimestampType)
			for j := 0; j < tsArray.Len(); j++ {
				if tsArray.IsNull(j) {
					values[idx] = nil
				} else {
					ts := tsArray.Value(j)
					var t time.Time
					switch tsType.Unit {
					case arrow.Nanosecond:
						t = time.Unix(0, int64(ts))
					case arrow.Microsecond:
						t = time.UnixMicro(int64(ts))
					case arrow.Millisecond:
						t = time.UnixMilli(int64(ts))
					case arrow.Second:
						t = time.Unix(int64(ts), 0)
					}
					if tsType.TimeZone != "" {
						if loc, err := time.LoadLocation(tsType.TimeZone); err == nil {
							t = t.In(loc)
						}
					}
					values[idx] = &t
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.FLOAT64:
		values := make([]*float64, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.Float64)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := arr.Value(j)
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.FLOAT32:
		values := make([]*float64, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.Float32)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := float64(arr.Value(j))
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.INT64:
		values := make([]*int64, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.Int64)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := arr.Value(j)
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.INT32:
		values := make([]*int64, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.Int32)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := int64(arr.Value(j))
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.INT16:
		values := make([]*int64, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.Int16)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := int64(arr.Value(j))
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.INT8:
		values := make([]*int64, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.Int8)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := int64(arr.Value(j))
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.UINT64:
		values := make([]*uint64, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.Uint64)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := arr.Value(j)
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.UINT32:
		values := make([]*uint64, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.Uint32)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := uint64(arr.Value(j))
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.BOOL:
		values := make([]*bool, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.Boolean)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := arr.Value(j)
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	case arrow.STRING, arrow.LARGE_STRING:
		values := make([]*string, numRows)
		idx := 0
		for _, chunk := range chunks {
			arr := chunk.(*array.String)
			for j := 0; j < arr.Len(); j++ {
				if arr.IsNull(j) {
					values[idx] = nil
				} else {
					v := arr.Value(j)
					values[idx] = &v
				}
				idx++
			}
		}
		return data.NewField(name, nil, values)

	default:
		// Fallback to string representation for unsupported types
		values := make([]string, numRows)
		idx := 0
		for _, chunk := range chunks {
			for j := 0; j < chunk.Len(); j++ {
				values[idx] = chunk.ValueStr(j)
				idx++
			}
		}
		return data.NewField(name, nil, values)
	}
}
