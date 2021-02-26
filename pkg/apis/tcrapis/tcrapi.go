/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2020 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package tcrapis

import (
	"errors"
	"net/http"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tcr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tcr/v20190924"
	"tkestack.io/image-transfer/configs"
	"tkestack.io/image-transfer/pkg/log"
)

// TCRAPIClient wrap http client
type TCRAPIClient struct {
	httpClient *http.Client
	url        string
}

// NewTCRAPIClient is new return *CCRAPIClient
func NewTCRAPIClient() *TCRAPIClient {
	httpclient := http.Client{}
	ai := TCRAPIClient{httpClient: &httpclient}

	return &ai
}

// GetAllNamespaceByName get all ns from tcr name
func (ai *TCRAPIClient) GetAllNamespaceByName(secret map[string]configs.Secret,
	region string, tcrName string) ([]string, string, error) {

	var nsList []string
	var tcrID string
	secretID, secretKey, err := GetTcrSecret(secret)

	if err != nil {
		log.Errorf("GetTcrSecret error: ", err)
		return nsList, tcrID, err
	}

	//get tcrId by tcr name
	filterValues := []string{tcrName}
	resp, err := ai.DescribeInstances(secretID, secretKey, region, 0, 100, "RegistryName", filterValues)
	if err != nil {
		log.Errorf("DescribeInstances error, ", err)
		return nsList, tcrID, err
	}

	tcrID = *resp.Response.Registries[0].RegistryId

	// tcr offset means page number, currently :(
	offset := int64(1)
	count := 0
	limit := int64(100)

	for {
		resp, err := ai.DescribeNamespaces(secretID, secretKey, region, offset, limit, tcrID)
		if err != nil {
			log.Errorf("DescribeNamespaces error, ", err)
			return nsList, tcrID, err
		}
		log.Infof("tcr namespace offset %d limit %d resp is %s", offset, limit, resp.ToJsonString())
		namespaceCount := *resp.Response.TotalCount
		count += len(resp.Response.NamespaceList)
		for _, ns := range resp.Response.NamespaceList {
			nsList = append(nsList, *ns.Name)
		}

		if int64(count) >= namespaceCount {
			break
		} else {
			offset += 1
		}

	}

	return nsList, tcrID, nil

}

// DescribeInstances is tcr api DescribeInstances
func (ai *TCRAPIClient) DescribeInstances(secretID, secretKey, region string, offset,
	limit int64, filterName string, filterValues []string) (*tcr.DescribeInstancesResponse, error) {

	credential := common.NewCredential(
		secretID,
		secretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tcr.tencentcloudapi.com"
	client, _ := tcr.NewClient(credential, region, cpf)

	request := tcr.NewDescribeInstancesRequest()

	request.Filters = []*tcr.Filter{
		{
			Values: common.StringPtrs(filterValues),
			Name:   common.StringPtr(filterName),
		},
	}

	response, err := client.DescribeInstances(request)

	if err != nil {
		log.Errorf("An error has returned: %s", err)
		return nil, err
	}

	return response, nil

}

// DescribeNamespaces is tcr api DescribeNamespaces
func (ai *TCRAPIClient) DescribeNamespaces(secretID, secretKey, region string, offset,
	limit int64, registryID string) (*tcr.DescribeNamespacesResponse, error) {

	credential := common.NewCredential(
		secretID,
		secretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tcr.tencentcloudapi.com"
	client, _ := tcr.NewClient(credential, region, cpf)

	request := tcr.NewDescribeNamespacesRequest()

	request.RegistryId = common.StringPtr(registryID)
	request.Limit = common.Int64Ptr(limit)
	request.Offset = common.Int64Ptr(offset)

	response, err := client.DescribeNamespaces(request)

	if err != nil {
		log.Errorf("An error has returned: %s", err)
		return nil, err
	}

	return response, nil

}

// CreateNamespace is tcr api CreateNamespace
func (ai *TCRAPIClient) CreateNamespace(secretID, secretKey, region string,
	registryID string, nsName string) (*tcr.CreateNamespaceResponse, error) {

	credential := common.NewCredential(
		secretID,
		secretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tcr.tencentcloudapi.com"
	client, _ := tcr.NewClient(credential, region, cpf)

	request := tcr.NewCreateNamespaceRequest()

	request.RegistryId = common.StringPtr(registryID)
	request.NamespaceName = common.StringPtr(nsName)
	request.IsPublic = common.BoolPtr(false)

	response, err := client.CreateNamespace(request)

	if err != nil {
		log.Errorf("An error has returned: %s", err)
		return nil, err
	}

	return response, nil

}

// GetTcrSecret get tcr secret from config
func GetTcrSecret(secret map[string]configs.Secret) (string, string, error) {
	var secretID string
	var secretKey string

	if tcr, ok := secret["tcr"]; ok {
		//tcr secret存在
		secretID = tcr.SecretID
		secretKey = tcr.SecretKey
	} else if ccr, ok := secret["ccr"]; ok {
		//用ccr secret代替tcr
		secretID = ccr.SecretID
		secretKey = ccr.SecretKey
	} else {
		return "", "", errors.New("no matched secret provided in secret file")
	}

	return secretID, secretKey, nil
}
