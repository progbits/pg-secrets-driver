package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	conn "github.com/progbits/pg-secrets-driver"
	"log"
	"net/url"
)

type AwsRdsSecret struct {
	Engine   string `json:"engine"`
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Dbname   string `json:"dbname"`
	Port     string `json:"port"`
}

type AwsSecretsManagerCredentialsProvider struct {
	ctx        context.Context
	secretName string
	retries    int
	count      int
	client     *secretsmanager.Client
}

func NewAwsSecretsManagerCredentialsProvider(ctx context.Context, client *secretsmanager.Client, secretName string) AwsSecretsManagerCredentialsProvider {
	return AwsSecretsManagerCredentialsProvider{
		ctx:        ctx,
		secretName: secretName,
		retries:    -1,
		client:     client,
	}
}

func (p *AwsSecretsManagerCredentialsProvider) GetDataSourceName() (string, error) {
	listSecretVersionIdsOutput, err := p.client.ListSecretVersionIds(
		p.ctx,
		&secretsmanager.ListSecretVersionIdsInput{
			SecretId: aws.String(p.secretName),
		},
	)
	if err != nil {
		log.Fatalf("unable to list secret versions, %v", err)
	}
	versionId := listSecretVersionIdsOutput.Versions[p.count].VersionId

	getSecretValueOutput, err := p.client.GetSecretValue(
		p.ctx,
		&secretsmanager.GetSecretValueInput{
			SecretId:  &p.secretName,
			VersionId: versionId,
		},
	)
	if err != nil {
		return "", err
	}

	p.count++
	secretString := *getSecretValueOutput.SecretString

	secret := AwsRdsSecret{}
	err = json.Unmarshal([]byte(secretString), &secret)
	if err != nil {
		log.Fatalf("failed to unmarshall secret, %v", err)
	}

	dsn, _ := url.Parse(
		fmt.Sprintf("postgresql://%s/%s", secret.Host, secret.Dbname),
	)
	query := dsn.Query()
	query.Add("user", secret.Username)
	query.Add("password", secret.Password)
	query.Add("sslmode", "disable")
	dsn.RawQuery = query.Encode()

	return dsn.String(), nil
}

func (p *AwsSecretsManagerCredentialsProvider) Retries() int {
	if p.retries > -1 {
		return p.retries
	}

	output, err := p.client.ListSecretVersionIds(
		p.ctx,
		&secretsmanager.ListSecretVersionIdsInput{
			SecretId: aws.String(p.secretName),
		},
	)
	if err != nil {
		log.Fatalf("unable to list secret versions, %v", err)
	}

	p.retries = len(output.Versions)
	return len(output.Versions)
}

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(fmt.Sprintf("failed to load configuration, %v", err))
	}

	client := secretsmanager.NewFromConfig(cfg)
	provider := NewAwsSecretsManagerCredentialsProvider(ctx, client, "PgSecretsDriverTest")

	connector := conn.NewPgSecretsConnector(&provider)

	db := sql.OpenDB(connector)
	_, err = db.Exec("SELECT 1")
	if err != nil {
		log.Print(err)
	}
}
