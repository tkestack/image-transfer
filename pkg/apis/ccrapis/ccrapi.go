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

package ccrapis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tcr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tcr/v20190924"
	"tkestack.io/image-transfer/configs"
	"tkestack.io/image-transfer/pkg/log"
	"tkestack.io/image-transfer/pkg/utils"
)

// CCRAPIClient wrap http client
type CCRAPIClient struct {
	httpClient *http.Client
	url        string
}

var regionPrefix = map[string]string{
	"ap-guangzhou":     "ccr",
	"ap-shanghai":      "ccr",
	"ap-nanjing":       "ccr",
	"ap-beijing":       "ccr",
	"ap-shenzhen":      "ccr",
	"ap-chongqing":     "ccr",
	"ap-chengdu":       "ccr",
	"ap-tianjin":       "ccr",
	"ap-hongkong":      "hkccr",
	"ap-shenzhen-fsi":  "szjrccr",
	"ap-shanghai-fsi":  "shjrccr",
	"ap-beijing-fsi":   "bjjrccr",
	"ap-singapore":     "sgccr",
	"ap-seoul":         "krccr",
	"ap-tokyo":         "jpccr",
	"ap-mumbai":        "inccr",
	"ap-bangkok":       "thccr",
	"na-toronto":       "caccr",
	"na-siliconvalley": "uswccr",
	"na-ashburn":       "useccr",
	"eu-frankfurt":     "deccr",
	"eu-moscow":        "ruccr",
}

// NewCCRAPIClient is new return *CCRAPIClient
func NewCCRAPIClient() *CCRAPIClient {
	httpclient := http.Client{}
	ai := CCRAPIClient{httpClient: &httpclient}

	return &ai
}

// GetAllNamespaceByName get all ns from ccr name
func (ai *CCRAPIClient) GetAllNamespaceByName(secret map[string]configs.Secret, region string) ([]string, error) {

	var nsList []string

	secretID, secretKey, err := GetCcrSecret(secret)

	if err != nil {
		log.Errorf("GetCcrSecret error: ", err)
		return nsList, err
	}

	offset := int64(0)
	count := 0
	limit := int64(100)

	for {
		resp, err := ai.DescribeNamespacePersonal(secretID, secretKey, region, offset, limit)
		if err != nil {
			log.Errorf("GetAllNamespaceByName error, ", err)
			return nsList, err
		}
		namespaceCount := *resp.Response.Data.NamespaceCount
		count += len(resp.Response.Data.NamespaceInfo)

		for _, ns := range resp.Response.Data.NamespaceInfo {
			nsList = append(nsList, *ns.Namespace)
		}

		if int64(count) >= namespaceCount {
			break
		} else {
			offset += limit
		}

	}

	return nsList, nil

}

//GenerateAllCcrRules generate all ccr rules
func (ai *CCRAPIClient) GenerateAllCcrRules(secret map[string]configs.Secret, ccrRegion string,
	failedNsList []string, tcrRegion string, tcrName string) (map[string]string, error) {

	rulesMap := make(map[string]string)

	secretID, secretKey, err := GetCcrSecret(secret)

	if err != nil {
		log.Errorf("GetCcrSecret error: ", err)
		return rulesMap, err
	}

	offset := int64(0)
	count := 0
	limit := int64(100)

	for {
		resp, err := ai.DescribeRepositoryOwnerPersonal(secretID, secretKey, ccrRegion, offset, limit)
		if err != nil {
			log.Errorf("get ccr repo error, ", err)
			return rulesMap, err
		}
		repoCount := *resp.Response.Data.TotalCount
		count += len(resp.Response.Data.RepoInfo)

		for _, repo := range resp.Response.Data.RepoInfo {
			ns := strings.Split(*repo.RepoName, "/")[0]
			if len(failedNsList) == 0 || !utils.IsContain(failedNsList, ns) {
				tags, err := ai.getRepoTags(secretID, secretKey, ccrRegion, *repo.RepoName)
				if err != nil {
					return nil, err
				}
				if len(tags) == 0 {
					continue
				}
				tagStr := strings.Join(tags, ",")
				source := fmt.Sprintf("%s%s%s:%s", regionPrefix[ccrRegion], ".ccs.tencentyun.com/", *repo.RepoName, tagStr)
				target := tcrName + ".tencentcloudcr.com/" + *repo.RepoName
				rulesMap[target] = source
			}
		}

		if int64(count) >= repoCount {
			break
		} else {
			offset += limit
		}

	}

	jsonStr, err := json.Marshal(rulesMap)
	if err != nil {
		log.Errorf("Marshal ccr rules map error %v, ", err)
	}
	go func() {
		err = ioutil.WriteFile("./ccr_to_tcr_rules", []byte(jsonStr), 0666)
		if err != nil {
			log.Errorf("WriteFile ccr rules error %v, ", err)
		}
	}()

	return rulesMap, nil

}

func (ai *CCRAPIClient) getRepoTags(secretID, secretKey, ccrRegion, repoName string) ([]string, error) {

	offset := int64(0)
	count := int64(0)
	limit := int64(100)

	var result []string

	for {
		resp, err := ai.DescribeImagePersonal(secretID, secretKey, ccrRegion, repoName, offset, limit)
		if err != nil {
			return nil, err
		}
		var tagCount int64

		if resp.Response != nil && resp.Response.Data != nil {
			tagCount = *resp.Response.Data.TagCount
		} else {
			return nil, errors.New("DescribeImagePersonal resp is nil")
		}
		if tagCount == 0 {
			return nil, nil
		}

		count += int64(len(resp.Response.Data.TagInfo))
		for _, tagInfo := range resp.Response.Data.TagInfo {
			result = append(result, *tagInfo.TagName)
		}

		if count >= tagCount {
			break
		} else {
			offset += limit
		}

	}

	return result, nil
}

func (ai *CCRAPIClient) DescribeImagePersonal(secretID, secretKey,
	region, repoName string, offset, limit int64) (*tcr.DescribeImagePersonalResponse, error) {

	credential := common.NewCredential(
		secretID,
		secretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tcr.tencentcloudapi.com"
	client, _ := tcr.NewClient(credential, region, cpf)

	request := tcr.NewDescribeImagePersonalRequest()

	request.Limit = common.Int64Ptr(limit)
	request.Offset = common.Int64Ptr(offset)
	request.RepoName = common.StringPtr(repoName)
	response, err := client.DescribeImagePersonal(request)

	if err != nil {
		log.Errorf("An error has returned: %s", err)
		return nil, err
	}

	return response, nil

}

// DescribeNamespacePersonal is ccr api DescribeNamespacePersonal
func (ai *CCRAPIClient) DescribeNamespacePersonal(secretID, secretKey,
	region string, offset, limit int64) (*tcr.DescribeNamespacePersonalResponse, error) {

	credential := common.NewCredential(
		secretID,
		secretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tcr.tencentcloudapi.com"
	client, _ := tcr.NewClient(credential, region, cpf)

	request := tcr.NewDescribeNamespacePersonalRequest()

	request.Namespace = common.StringPtr("")
	request.Limit = common.Int64Ptr(limit)
	request.Offset = common.Int64Ptr(offset)

	response, err := client.DescribeNamespacePersonal(request)

	if err != nil {
		log.Errorf("An error has returned: %s", err)
		return nil, err
	}

	return response, nil

}

// DescribeRepositoryOwnerPersonal is ccr api DescribeRepositoryOwnerPersonal
func (ai *CCRAPIClient) DescribeRepositoryOwnerPersonal(secretID, secretKey,
	region string, offset, limit int64) (*tcr.DescribeRepositoryOwnerPersonalResponse, error) {

	credential := common.NewCredential(
		secretID,
		secretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tcr.tencentcloudapi.com"
	client, _ := tcr.NewClient(credential, region, cpf)

	request := tcr.NewDescribeRepositoryOwnerPersonalRequest()

	request.Limit = common.Int64Ptr(limit)
	request.Offset = common.Int64Ptr(offset)

	response, err := client.DescribeRepositoryOwnerPersonal(request)

	if err != nil {
		log.Errorf("An error has returned: %s", err)
		return nil, err
	}

	return response, nil

}

// GetCcrSecret get ccr secret from configs
func GetCcrSecret(secret map[string]configs.Secret) (string, string, error) {
	var secretID string
	var secretKey string

	if ccr, ok := secret["ccr"]; ok {
		//ccr secret存在
		secretID = ccr.SecretID
		secretKey = ccr.SecretKey
	} else if tcr, ok := secret["tcr"]; ok {
		//用tcr secret代替ccr
		secretID = tcr.SecretID
		secretKey = tcr.SecretKey
	} else {
		return "", "", errors.New("no matched secret provided in secret file")
	}

	return secretID, secretKey, nil
}
