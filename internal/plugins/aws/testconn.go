package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func testAWSCreds(p credsInput) error {
	prov := credentials.NewStaticCredentialsProvider(p.AccessKey, p.SecretKey, "")
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(p.Region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(prov)),
	)
	if err != nil {
		return err
	}

	client := sts.NewFromConfig(cfg)
	if _, err := client.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{}); err != nil {
		return fmt.Errorf("STS GetCallerIdentity failed: %w", err)
	}

	fmt.Println("✅ AWS credentials are valid")
	return nil
}
