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

package configs

import (
	"errors"
	"fmt"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v2"
	"os"
	"strings"
	"sync"
	"tkestack.io/image-transfer/pkg/image-transfer/options"
	"tkestack.io/image-transfer/pkg/log"
)


var (
	once     sync.Once
	instance *Configs
	//QPS rateLimit qps
	QPS int
	//sessionInstance *SessionConfigs
)

const (
	maxRatelimit int = 30000
	maxRoutineNums int = 50
)

// Configs struct save of all config
type Configs struct {
	FlagConf *options.ClientOptions
	Conf     *ini.File
	Security      map[string]Security
	ImageList map[string]string
	Secret map[string]Secret
	//ConfMap       map[string]interface{}
	//ConfMapString map[string]string
}

// Security describes the authentication information of a registry
type Security struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Insecure bool   `json:"insecure" yaml:"insecure"`
}

// Secret describes secret info for tencent cloud
type Secret struct {
	SecretID string `json:"secretId" yaml:"secretId"`
	SecretKey string `json:"secretKey" yaml:"secretKey"`
}


// InitConfigs InitLogger initializes logger the way we want for tke.
func InitConfigs(opts *options.ClientOptions) (*Configs, error) {
	//log.Println(opts.Config.ConfigFile)
	once.Do(func() {
		instance = &Configs{}

		instance.FlagConf = opts
	})

	if instance.FlagConf.Config.CCRToTCR == true {
		if len(instance.FlagConf.Config.SecretFile) == 0 || len(instance.FlagConf.Config.SecurityFile) == 0 {
			return nil, errors.New("no SecretFile or security file is provided, Exit")
		} else if len(instance.FlagConf.Config.TCRName) == 0 {
			return nil, errors.New("no tcr name is provided, Exit")
		} else {
			secret, err := instance.GetSecret()
			if err != nil {
				return nil, err
			}
			instance.Secret = secret
			securityList, err := instance.GetSecurity()
			if err != nil {
				return nil, err
			}
			instance.Security = securityList
		}
	} else {
		if len(instance.FlagConf.Config.RuleFile) == 0 || len(instance.FlagConf.Config.SecurityFile) == 0 {
			return nil, errors.New("no rule file or security file is provided, Exit")
		}
		instance.ImageList = instance.GetImageList()

		securityList, err := instance.GetSecurity()
		if err != nil {
			return nil, err
		}
		instance.Security = securityList


	}

	if instance.FlagConf.Config.RoutineNums > maxRoutineNums {
		instance.FlagConf.Config.RoutineNums = maxRoutineNums
	}

	if instance.FlagConf.Config.QPS > maxRatelimit {
		instance.FlagConf.Config.QPS = maxRatelimit
	}

	QPS = instance.FlagConf.Config.QPS


	return instance, nil
}

// GetConfigs get config of Configs instance
func GetConfigs() *Configs {
	/*if instance == nil{
		log.Fatalf("Fail to get instance: %v", instance)
	}*/
	return instance
}

// GetImageList get images list of configs instance
func (c *Configs) GetImageList() map[string]string {
	var imageList map[string]string


	if err := openAndDecode(c.FlagConf.Config.RuleFile, &imageList); err != nil {
		log.Errorf("decode config file %v error: %v", c.FlagConf.Config.RuleFile, err)
		return nil
	}


	return imageList
}

// GetSecurity gets the Security information in Config
func (c *Configs) GetSecurity() (map[string]Security, error) {
	var securityList map[string]Security

	if err := openAndDecode(c.FlagConf.Config.SecurityFile, &securityList); err != nil {
		log.Errorf("decode config file %v error: %v", c.FlagConf.Config.SecurityFile, err)
		return securityList, err
	}


	return securityList, nil
}


// GetSecuritySpecific gets the specific authentication information in Config
func (c *Configs) GetSecuritySpecific(registry string, namespace string) (Security, bool) {

	// key of each AuthList item can be "registry/namespace" or "registry" only
	registryAndNamespace := registry + "/" + namespace

	if moreSpecificAuth, exist := c.Security[registryAndNamespace]; exist {
		return moreSpecificAuth, exist
	}
	auth, exist := c.Security[registry]
	return auth, exist
}

// GetSecret get secret from secret file
func (c *Configs) GetSecret() (map[string]Secret, error) {
	var secret map[string]Secret

	if err := openAndDecode(c.FlagConf.Config.SecretFile, &secret); err != nil {
		log.Errorf("decode secret file %v error: %v", c.FlagConf.Config.SecretFile, err)
		return secret, err
	}

	return secret, nil

}


// Open yaml file and decode into target interface
func openAndDecode(filePath string, target interface{}) error {
	if !strings.HasSuffix(filePath, ".yaml") {
		return fmt.Errorf("only support yaml format file")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file %v not exist: %v", filePath, err)
	}

	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		return fmt.Errorf("open file %v error: %v", filePath, err)
	}


	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("unmarshal config error: %v", err)
	}


	return nil
}
