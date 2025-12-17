package azureclient

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type StaticTokenCredential struct {
	token string
}

func NewStaticTokenCredential(token string) *StaticTokenCredential {
	return &StaticTokenCredential{
		token: token,
	}
}

func (c *StaticTokenCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	expiresOn := time.Now().Add(1 * time.Hour)
	return azcore.AccessToken{
		Token:     c.token,
		ExpiresOn: expiresOn,
	}, nil
}
