package parquet

import (
	"context"
	"fmt"
	"io"

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
	// We iterate over columns and create fields.

	frame := data.NewFrame(key)
	for i := 0; i < int(table.NumCols()); i++ {
		col := table.Column(i)
		fieldName := table.Schema().Field(i).Name

		// Create a string field for now to satisfy the prototype and avoid panics
		values := make([]string, table.NumRows())
		for j := 0; j < int(table.NumRows()); j++ {
			// Get string representation of the value
			// col.Data().Chunk(0) might not be the only chunk, but for simple files it is.
			// Reaching into the first chunk for now.
			if col.Len() > 0 {
				values[j] = col.Data().Chunk(0).ValueStr(j)
			}
		}
		frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
	}

	return []*data.Frame{frame}, nil
}
