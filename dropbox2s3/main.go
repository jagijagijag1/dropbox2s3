package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
)

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context) error {
	// extract env var
	dropboxToken := os.Getenv("DROPBOX_TOKEN")
	imgFolderPath := os.Getenv("IMG_FOLDER_PATH")
	bucketName := os.Getenv("BUCKET_NAME")

	// tansport image from dropbox to s3
	transport(dropboxToken, imgFolderPath, bucketName)

	return nil
}

func transport(dropboxToken, imgFolderPath, bucketName string) {
	// dropbox setting
	config := dropbox.Config{
		Token: dropboxToken,
	}
	dbx := files.New(config)

	// get info under the imgFolderPath folder
	arg := files.NewListFolderArg(imgFolderPath)
	resp, err := dbx.ListFolder(arg)
	if err != nil {
		fmt.Println("err in accesing dropbox folder")
		fmt.Println(err)
		return
	}

	// for each file/folder
	for _, e := range resp.Entries {
		// use type annotation to cast e (IsMetadata) to file (FileMetadata)
		f, ok := e.(*files.FileMetadata)
		if ok {
			// if e is file, download content
			fmt.Println("find file ", f.Name)
			fpath := f.Metadata.PathLower
			dlArg := files.NewDownloadArg(fpath)
			res, content, err := dbx.Download(dlArg)
			if err != nil {
				fmt.Println("err in downloading file")
				fmt.Println(err)
				return
			}

			// and then put it to S3
			err = putToS3(bucketName, res.Metadata.Name, content)
			if err == nil {
				// if successed, remove transported file on dropbox
				deleteFromDropbox(dbx, fpath)
			}
		}
	}
}

func putToS3(bucketName, fileName string, content io.ReadCloser) error {
	// create s3 client
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"),
	}))
	svc := s3.New(sess)

	// create put-object input
	blob, _ := ioutil.ReadAll(content)
	f := bytes.NewReader(blob)
	rs := aws.ReadSeekCloser(f)
	input := &s3.PutObjectInput{
		Body:   rs,
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
	}

	// put contet to s3
	_, err := svc.PutObject(input)
	if err != nil {
		fmt.Println("err in putting object")
		fmt.Println(err)
		return err
	}

	fmt.Println("put object ", fileName, " to ", bucketName)
	return nil
}

func deleteFromDropbox(dbx files.Client, filepath string) {
	delArg := files.NewDeleteArg(filepath)
	_, err := dbx.Delete(delArg)
	if err != nil {
		fmt.Println("err in downloading file")
		fmt.Println(err)
		return
	}
}

func main() {
	lambda.Start(Handler)
}
