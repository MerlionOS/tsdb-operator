// Command tsdb-ctl is a small CLI companion to tsdb-operator. Today it
// handles "restore" — pulling a snapshot back from S3 to local disk so it
// can be staged into a PVC via kubectl cp.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "restore":
		if err := restore(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "list":
		if err := list(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `tsdb-ctl — companion CLI for tsdb-operator

Commands:
  list     List available snapshots in S3
  restore  Download a snapshot from S3 to a local directory

Run 'tsdb-ctl <command> -h' for command flags.`)
}

type s3Flags struct {
	endpoint, region, bucket, prefix string
}

func (f *s3Flags) bind(fs *flag.FlagSet) {
	fs.StringVar(&f.endpoint, "endpoint", "", "Override S3 endpoint (e.g. MinIO URL).")
	fs.StringVar(&f.region, "region", "us-east-1", "AWS region.")
	fs.StringVar(&f.bucket, "bucket", "", "S3 bucket (required).")
	fs.StringVar(&f.prefix, "prefix", "", "S3 key prefix to scope the listing.")
}

func (f *s3Flags) client(ctx context.Context) (*s3.Client, error) {
	if f.bucket == "" {
		return nil, fmt.Errorf("--bucket is required")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(f.region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		if f.endpoint != "" {
			o.BaseEndpoint = aws.String(f.endpoint)
			o.UsePathStyle = true
		}
	}), nil
}

func list(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	var flags s3Flags
	flags.bind(fs)
	_ = fs.Parse(args)
	ctx := context.Background()
	client, err := flags.client(ctx)
	if err != nil {
		return err
	}
	out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(flags.bucket),
		Prefix: aws.String(flags.prefix),
	})
	if err != nil {
		return fmt.Errorf("list objects: %w", err)
	}
	type entry struct {
		key   string
		size  int64
		stamp string
	}
	var entries []entry
	for _, o := range out.Contents {
		entries = append(entries, entry{
			key:   aws.ToString(o.Key),
			size:  aws.ToInt64(o.Size),
			stamp: aws.ToTime(o.LastModified).Format("2006-01-02T15:04:05Z"),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].stamp > entries[j].stamp })
	for _, e := range entries {
		fmt.Printf("%s  %10d  %s\n", e.stamp, e.size, e.key)
	}
	return nil
}

func restore(args []string) error {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	var flags s3Flags
	flags.bind(fs)
	key := fs.String("key", "", "Exact S3 key to download. If empty, the newest object under --prefix is chosen.")
	dest := fs.String("dest", "./restore", "Local directory to write the snapshot into.")
	_ = fs.Parse(args)

	ctx := context.Background()
	client, err := flags.client(ctx)
	if err != nil {
		return err
	}

	resolved := *key
	if resolved == "" {
		latest, err := latestKey(ctx, client, flags.bucket, flags.prefix)
		if err != nil {
			return err
		}
		if latest == "" {
			return fmt.Errorf("no objects under s3://%s/%s", flags.bucket, flags.prefix)
		}
		resolved = latest
	}

	if err := os.MkdirAll(*dest, 0o755); err != nil {
		return fmt.Errorf("mkdir dest: %w", err)
	}
	local := filepath.Join(*dest, filepath.Base(resolved))
	f, err := os.Create(local)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	get, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(flags.bucket),
		Key:    aws.String(resolved),
	})
	if err != nil {
		return fmt.Errorf("get object: %w", err)
	}
	defer func() { _ = get.Body.Close() }()

	n, err := io.Copy(f, get.Body)
	if err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	fmt.Printf("downloaded s3://%s/%s → %s (%d bytes)\n", flags.bucket, resolved, local, n)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. kubectl cp", local, "<ns>/<prometheus-pod>:/prometheus/")
	tarName := filepath.Base(resolved)
	fmt.Printf("  2. kubectl exec -n <ns> <prometheus-pod> -- tar -xf /prometheus/%s -C /prometheus/\n", tarName)
	fmt.Println("  3. Delete the pod so the StatefulSet reschedules with restored data.")
	return nil
}

func latestKey(ctx context.Context, client *s3.Client, bucket, prefix string) (string, error) {
	out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return "", fmt.Errorf("list objects: %w", err)
	}
	var newest string
	var newestStamp string
	for _, o := range out.Contents {
		stamp := aws.ToTime(o.LastModified).Format("2006-01-02T15:04:05Z")
		if strings.Compare(stamp, newestStamp) > 0 {
			newestStamp = stamp
			newest = aws.ToString(o.Key)
		}
	}
	return newest, nil
}
