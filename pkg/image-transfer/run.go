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

package imagetransfer

import (
	"container/list"
	"fmt"
	"strings"
	"sync"
	"time"

	"tkestack.io/image-transfer/configs"
	"tkestack.io/image-transfer/pkg/apis/ccrapis"
	"tkestack.io/image-transfer/pkg/apis/tcrapis"
	"tkestack.io/image-transfer/pkg/image-transfer/options"
	"tkestack.io/image-transfer/pkg/log"
	"tkestack.io/image-transfer/pkg/transfer"
	"tkestack.io/image-transfer/pkg/utils"
)

//Client is a transfer client
type Client struct {
	// a Transfer.Job list
	jobList *list.List

	// a URLPair list store origin url pair from rule file
	urlPairList *list.List
	// store standard image url
	normalURLPairList *list.List

	// failed list
	failedJobList              *list.List
	failedJobGenerateList      *list.List
	failedGenNormalURLPairList *list.List

	config *configs.Configs

	//finished generate ccrToTcr urlPair
	urlPairFinished bool
	// mutex
	jobListMutex                    sync.Mutex
	urlPairListMutex                sync.Mutex
	normalURLPairListMutex          sync.Mutex
	failedJobListMutex              sync.Mutex
	failedJobGenerateListMutex      sync.Mutex
	failedGenNormalURLPairListMutex sync.Mutex
	urlPairFinishedMutex            sync.Mutex
}

// URLPair is a pair of source and target url
type URLPair struct {
	source string
	target string
}

// Run is main function of a transfer client
func (c *Client) Run() error {

	if c.config.FlagConf.Config.CCRToTCR {
		return c.CCRToTCRTransfer()
	}

	return c.NormalTransfer(c.config.ImageList, nil, nil, nil)

}

//CCRToTCRTransfer transfer ccr to tcr
func (c *Client) CCRToTCRTransfer() error {

	ccrClient := ccrapis.NewCCRAPIClient()
	ccrNs, err := ccrClient.GetAllNamespaceByName(c.config.Secret, c.config.FlagConf.Config.CCRRegion)
	log.Debugf("ccr namespaces is %s", ccrNs)
	if err != nil {
		log.Errorf("Get ccr ns returned error: %s", err)
		return err
	}

	tcrClient := tcrapis.NewTCRAPIClient()
	tcrNs, tcrID, err := tcrClient.GetAllNamespaceByName(c.config.Secret,
		c.config.FlagConf.Config.TCRRegion, c.config.FlagConf.Config.TCRName)
	log.Debugf("tcr namespaces is %s", tcrNs)
	if err != nil {
		log.Errorf("Get tcr ns returned error: %s", err)
		return err
	}

	//create ccr ns in tcr
	failedNsList, err := c.CreateTcrNs(tcrClient, ccrNs, tcrNs, c.config.Secret, c.config.FlagConf.Config.TCRRegion, tcrID)
	if err != nil {
		log.Errorf("CreateTcrNs error: %s", err)
		return err
	}

	//retry failedNsList
	if len(failedNsList) != 0 {
		log.Infof("some ccr namespace create failed in tcr, retry Create Tcr Ns.")
		for times := 0; times < c.config.FlagConf.Config.RetryNums && len(failedNsList) != 0; times++ {
			tmpFailedNsList, err := c.RetryCreateTcrNs(tcrClient, failedNsList,
				c.config.Secret, c.config.FlagConf.Config.TCRRegion)
			if err != nil {
				continue
			} else {
				failedNsList = tmpFailedNsList
			}
		}
	}

	if len(failedNsList) != 0 {
		log.Warnf("some ccr namespace create failed in tcr: %s", failedNsList)
	}

	//generate transfer rules
	repoChan, err := c.GenerateCcrToTcrRules(failedNsList, ccrClient, c.config.Secret, c.config.FlagConf.Config.CCRRegion,
		c.config.FlagConf.Config.TCRRegion, c.config.FlagConf.Config.TCRName)
	if err != nil {
		return err
	}

	return c.NormalTransfer(nil, ccrClient, tcrClient, repoChan)

}

//GenerateCcrToTcrRules generate rules of ccr transfer to tcr
func (c *Client) GenerateCcrToTcrRules(failedNsList []string, ccrClient *ccrapis.CCRAPIClient,
	secret map[string]configs.Secret, ccrRegion string, tcrRegion string, tcrName string) (chan string, error) {

	secretID, secretKey, err := ccrapis.GetCcrSecret(secret)
	if err != nil {
		log.Errorf("GetCcrSecret error: %s", err)
		return nil, err
	}
	resp, err := ccrClient.DescribeRepositoryOwnerPersonal(secretID, secretKey, ccrRegion, 0, 1)
	if err != nil {
		log.Errorf("get ccr repo count error, %s", err)
		return nil, fmt.Errorf("get ccr repo count error %s", err)
	}
	totalRepo := *resp.Response.Data.TotalCount
	log.Debugf("total repo is %d", totalRepo)
	repoChan := make(chan string, totalRepo)
	err = ccrClient.GetAllCcrRepo(secret, ccrRegion, failedNsList, tcrRegion, tcrName, repoChan, totalRepo)
	if err != nil {
		log.Errorf("get ccr repo to tcr rules failed: %s", err)
		return nil, err
	}

	return repoChan, nil

}

//RetryCreateTcrNs retry to create tcr namespaces
func (c *Client) RetryCreateTcrNs(tcrClient *tcrapis.TCRAPIClient, retryList []string,
	secret map[string]configs.Secret, region string) ([]string, error) {
	var failedList []string

	secretID, secretKey, err := tcrapis.GetTcrSecret(secret)
	if err != nil {
		log.Errorf("GetTcrSecret error: %s", err)
		return failedList, err
	}

	tcrNs, tcrID, err := tcrClient.GetAllNamespaceByName(c.config.Secret,
		c.config.FlagConf.Config.TCRRegion, c.config.FlagConf.Config.TCRName)
	log.Debugf("tcr namespaces is %s", tcrNs)
	if err != nil {
		log.Errorf("retry create tcr ns, get tcr ns error: ", err)
		return nil, err
	}

	for _, ns := range retryList {
		if !utils.IsContain(tcrNs, ns) {
			_, err := tcrClient.CreateNamespace(secretID, secretKey, region, tcrID, ns)
			if err != nil {
				log.Errorf("tcr CreateNamespace %s error: %s", ns, err)
				failedList = append(failedList, ns)
			}
		}
	}

	return failedList, nil

}

//CreateTcrNs create tcr namespaces
func (c *Client) CreateTcrNs(tcrClient *tcrapis.TCRAPIClient, ccrNs, tcrNs []string,
	secret map[string]configs.Secret, region string, tcrID string) ([]string, error) {

	var failedList []string

	secretID, secretKey, err := tcrapis.GetTcrSecret(secret)

	if err != nil {
		log.Errorf("GetTcrSecret error: %s", err)
		return failedList, err
	}

	for _, ns := range ccrNs {
		if !utils.IsContain(tcrNs, ns) {
			log.Infof("create namespace %s", ns)
			_, err := tcrClient.CreateNamespace(secretID, secretKey, region, tcrID, ns)
			if err != nil {
				log.Errorf("tcr CreateNamespace %s error:  %s", ns, err)
				failedList = append(failedList, ns)
			}
		}
	}

	return failedList, nil

}

//NormalTransfer is the normal mode of transfer
func (c *Client) NormalTransfer(imageList map[string]string, ccrClient *ccrapis.CCRAPIClient, tcrClient *tcrapis.TCRAPIClient, repoChan chan string) error {
	jobListChan := make(chan *transfer.Job, c.config.FlagConf.Config.RoutineNums)
	fmt.Println("Start to handle transfer jobs, please wait ...")
	wg := sync.WaitGroup{}

	// generate goroutines to handle transfer jobs
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.jobsHandler(jobListChan)
	}()

	// ccrToTcr progress is (ccr api)repo --> repochan --> NormalPairList --> jobListChan
	if c.config.FlagConf.Config.CCRToTCR {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.HandleCcrToTCrTags(repoChan)
			c.SetURLPairFinished()
		}()
	} else {
		// Normal progress is urlPairList --> NormalPairList --> jobListChan
		for source, target := range imageList {
			c.urlPairList.PushBack(&URLPair{
				source: source,
				target: target,
			})
		}

		// carry urlPair to NormalURLPair
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.HandleURLPair()
			c.SetURLPairFinished()
		}()
	}

	// get item from NormalURLPair to jobListChan
	c.rulesHandler(jobListChan)

	wg.Wait()

	log.Infof("Start to retry failed jobs...")

	for times := 0; times < c.config.FlagConf.Config.RetryNums; times++ {
		log.Debugf("failedjobList len %d, failedGenNormalURLPairList len %d, failedJobGenerateList len %d", c.failedJobList.Len(), c.failedGenNormalURLPairList.Len(), c.failedJobGenerateList.Len())
		c.Retry()
	}

	if c.failedJobList.Len() != 0 {
		log.Infof("################# %v failed transfer jobs: #################", c.failedJobList.Len())
		for e := c.failedJobList.Front(); e != nil; e = e.Next() {
			log.Infof(e.Value.(*transfer.Job).Source.GetRegistry() + "/" +
				e.Value.(*transfer.Job).Source.GetRepository() + ":" + e.Value.(*transfer.Job).Source.GetTag())

		}
	}

	if c.failedGenNormalURLPairList.Len() != 0 {
		log.Infof("################# %v failed generate Normal urlPair: #################", c.failedGenNormalURLPairList.Len())
		for e := c.failedGenNormalURLPairList.Front(); e != nil; e = e.Next() {
			log.Infof(e.Value.(*URLPair).source + ": " + e.Value.(*URLPair).target)

		}
	}

	if c.failedJobGenerateList.Len() != 0 {
		log.Infof("################# %v failed generate jobs: #################", c.failedJobGenerateList.Len())
		for e := c.failedJobGenerateList.Front(); e != nil; e = e.Next() {
			log.Infof(e.Value.(*URLPair).source + ": " + e.Value.(*URLPair).target)

		}
	}

	log.Infof("################# Finished, %v transfer jobs failed, %v normal urlPair generate failed, %v jobs generate failed #################",
		c.failedJobList.Len(), c.failedGenNormalURLPairList.Len(), c.failedJobGenerateList.Len())

	return nil

}

//Retry is retry the failed job
func (c *Client) Retry() {
	retryJobListChan := make(chan *transfer.Job, c.config.FlagConf.Config.RoutineNums)

	wg1 := sync.WaitGroup{}
	wg1.Add(1)
	go func() {
		defer func() {
			wg1.Done()
		}()
		c.jobsHandler(retryJobListChan)
	}()

	if c.failedJobList.Len() != 0 {
		failedJobListChan := make(chan *transfer.Job, c.failedJobList.Len())
		for {
			failedJob := c.failedJobList.Front()
			if failedJob == nil {
				break
			}
			log.Infof("put failed job to failedJobListChan %s/%s:%s %s/%s:%s", failedJob.Value.(*transfer.Job).Source.GetRegistry(), failedJob.Value.(*transfer.Job).Source.GetRepository(), failedJob.Value.(*transfer.Job).Source.GetTag(), failedJob.Value.(*transfer.Job).Target.GetRegistry(), failedJob.Value.(*transfer.Job).Target.GetRepository(), failedJob.Value.(*transfer.Job).Target.GetTag())
			failedJobListChan <- failedJob.Value.(*transfer.Job)
			c.failedJobList.Remove(failedJob)
		}
		close(failedJobListChan)
		wg1.Add(1)
		go func() {
			defer wg1.Done()
			c.jobsHandler(failedJobListChan)
		}()
	}

	if c.failedGenNormalURLPairList.Len() != 0 || c.failedJobGenerateList.Len() != 0 {
		if c.failedGenNormalURLPairList.Len() != 0 {
			c.urlPairList.PushBackList(c.failedGenNormalURLPairList)
			c.failedGenNormalURLPairList.Init()
			if !c.config.FlagConf.Config.CCRToTCR {
				c.HandleURLPair()
			} else {
				c.CcrtoTcrGenTagRetry()
			}
		}

		if c.failedJobGenerateList.Len() != 0 {
			c.normalURLPairList.PushBackList(c.failedJobGenerateList)
			c.failedJobGenerateList.Init()
		}
		c.rulesHandler(retryJobListChan)
	} else {
		close(retryJobListChan)
	}

	wg1.Wait()
}

// NewTransferClient creates a transfer client
func NewTransferClient(opts *options.ClientOptions) (*Client, error) {

	clientConfig, err := configs.InitConfigs(opts)

	if err != nil {
		return nil, err
	}

	return &Client{
		jobList:                         list.New(),
		urlPairList:                     list.New(),
		failedJobList:                   list.New(),
		failedJobGenerateList:           list.New(),
		normalURLPairList:               list.New(),
		failedGenNormalURLPairList:      list.New(),
		config:                          clientConfig,
		jobListMutex:                    sync.Mutex{},
		urlPairListMutex:                sync.Mutex{},
		failedJobListMutex:              sync.Mutex{},
		failedJobGenerateListMutex:      sync.Mutex{},
		urlPairFinishedMutex:            sync.Mutex{},
		failedGenNormalURLPairListMutex: sync.Mutex{},
	}, nil
}

func (c *Client) rulesHandler(jobListChan chan *transfer.Job) {
	defer func() {
		close(jobListChan)
	}()

	routineNum := c.config.FlagConf.Config.RoutineNums
	wg := sync.WaitGroup{}

	for i := 0; i < routineNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer log.Debugf("exit rule handler main loop")
			for {
				urlPair, empty := c.GetNormalURLPair()
				// no more job to generate
				if empty && c.IsURLPairFinished() {
					log.Debugf("url pair is empty")
					break
				}
				if empty {
					log.Debugf("not finieshed but url pair is empty")
					time.Sleep(100 * time.Millisecond)
					continue
				}
				log.Infof("generate job source %s, target %s", urlPair.source, urlPair.target)
				err := c.GenerateTransferJob(jobListChan, urlPair.source, urlPair.target)
				if err != nil {
					log.Errorf("Generate transfer job %s to %s error: %v", urlPair.source, urlPair.target, err)
					// put to failedJobGenerateList
					c.PutAFailedURLPair(urlPair)
				}
			}
		}()
	}
	wg.Wait()
}

func (c *Client) jobsHandler(jobListChan chan *transfer.Job) {

	routineNum := c.config.FlagConf.Config.RoutineNums
	wg := sync.WaitGroup{}
	for i := 0; i < routineNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				job, ok := <-jobListChan
				if !ok {
					break
				}
				if err := job.Run(); err != nil {
					log.Errorf("handle job failed %s/%s:%s, %s", job.Source.GetRegistry(), job.Source.GetRepository(), job.Source.GetTag(), err)
					c.PutAFailedJob(job)
				}
			}
		}()
	}

	wg.Wait()

}

// GetURLPair gets a URLPair from urlPairList
func (c *Client) GetURLPair() (*URLPair, bool) {
	c.urlPairListMutex.Lock()
	defer func() {
		c.urlPairListMutex.Unlock()
	}()

	urlPair := c.urlPairList.Front()
	if urlPair == nil {
		return nil, true
	}
	c.urlPairList.Remove(urlPair)

	return urlPair.Value.(*URLPair), false
}

// PutURLPair puts a URLPair to urlPairList
func (c *Client) PutURLPair(urlPair *URLPair) {
	c.urlPairListMutex.Lock()
	defer c.urlPairListMutex.Unlock()
	c.urlPairList.PushBack(urlPair)
}

// GetNormalURLPair gets a URLPair from normalurlPairList
func (c *Client) GetNormalURLPair() (*URLPair, bool) {
	c.normalURLPairListMutex.Lock()
	defer func() {
		c.normalURLPairListMutex.Unlock()
	}()

	urlPair := c.normalURLPairList.Front()
	if urlPair == nil {
		return nil, true
	}
	c.normalURLPairList.Remove(urlPair)

	return urlPair.Value.(*URLPair), false
}

// PutNormalURLPair puts a URLPair to normalurlPairList
func (c *Client) PutNormalURLPair(urlPair *URLPair) {
	c.normalURLPairListMutex.Lock()
	defer c.normalURLPairListMutex.Unlock()
	c.normalURLPairList.PushBack(urlPair)
}

// GetJob return a transfer.Job struct if the job list is not empty
func (c *Client) GetJob() (*transfer.Job, bool) {
	c.jobListMutex.Lock()
	defer func() {
		c.jobListMutex.Unlock()
	}()

	job := c.jobList.Front()
	if job == nil {
		return nil, true
	}
	c.jobList.Remove(job)

	return job.Value.(*transfer.Job), false
}

// PutJob puts a transfer.Job struct to job list
func (c *Client) PutJob(job *transfer.Job) {
	c.jobListMutex.Lock()
	defer func() {
		c.jobListMutex.Unlock()
	}()

	if c.jobList != nil {
		c.jobList.PushBack(job)
	}
}

// GenerateTransferJob creates transfer jobs from normalURLPair
func (c *Client) GenerateTransferJob(jobListChan chan *transfer.Job, source string, target string) error {
	if source == "" {
		return fmt.Errorf("source url should not be empty")
	}

	sourceURL, err := utils.NewRepoURL(source)
	if err != nil {
		return fmt.Errorf("url %s format error: %v", source, err)
	}

	if target == "" {
		return fmt.Errorf("target url should not be empty")
	}

	targetURL, err := utils.NewRepoURL(target)
	if err != nil {
		return fmt.Errorf("url %s format error: %v", target, err)
	}

	// if tag is not specific
	if sourceURL.GetTag() == "" {
		return fmt.Errorf("source tag empty, source: %s", sourceURL.GetURL())
	}

	if targetURL.GetTag() == "" {
		return fmt.Errorf("target tag empty, target: %s", targetURL.GetURL())
	}

	var imageSource *transfer.ImageSource
	var imageTarget *transfer.ImageTarget

	sourceSecurity, exist := c.config.GetSecuritySpecific(sourceURL.GetRegistry(), sourceURL.GetNamespace())
	if exist {
		log.Infof("Find auth information for %v, username: %v", sourceURL.GetURL(), sourceSecurity.Username)
	} else {
		log.Infof("Cannot find auth information for %v, pull actions will be anonymous", sourceURL.GetURL())
	}

	targetSecurity, exist := c.config.GetSecuritySpecific(targetURL.GetRegistry(), targetURL.GetNamespace())
	if exist {
		log.Infof("Find auth information for %v, username: %v", targetURL.GetURL(), targetSecurity.Username)

	} else {
		log.Infof("Cannot find auth information for %v, push actions will be anonymous", targetURL.GetURL())
	}

	imageSource, err = transfer.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(), sourceURL.GetTag(), sourceSecurity.Username, sourceSecurity.Password, sourceSecurity.Insecure)
	if err != nil {
		return fmt.Errorf("generate %s image source error: %v", sourceURL.GetURL(), err)
	}

	imageTarget, err = transfer.NewImageTarget(targetURL.GetRegistry(), targetURL.GetRepoWithNamespace(), targetURL.GetTag(), targetSecurity.Username, targetSecurity.Password, targetSecurity.Insecure)
	if err != nil {
		return fmt.Errorf("generate %s image target error: %v", sourceURL.GetURL(), err)
	}

	jobListChan <- transfer.NewJob(imageSource, imageTarget)

	log.Infof("Generate a job for %s to %s", sourceURL.GetURL(), targetURL.GetURL())
	return nil
}

// GetFailedJob gets a failed job from failedJobList
func (c *Client) GetFailedJob() (*transfer.Job, bool) {
	c.failedJobListMutex.Lock()
	defer func() {
		c.failedJobListMutex.Unlock()
	}()

	failedJob := c.failedJobList.Front()
	if failedJob == nil {
		return nil, true
	}
	c.failedJobList.Remove(failedJob)

	return failedJob.Value.(*transfer.Job), false
}

// PutAFailedJob puts a failed job to failedJobList
func (c *Client) PutAFailedJob(failedJob *transfer.Job) {

	c.failedJobListMutex.Lock()
	defer func() {
		c.failedJobListMutex.Unlock()
	}()

	if c.failedJobList != nil {
		c.failedJobList.PushBack(failedJob)
	}
}

// GetAFailedURLPair get a URLPair from failedJobGenerateList
func (c *Client) GetAFailedURLPair() (*URLPair, bool) {
	c.failedJobGenerateListMutex.Lock()
	defer func() {
		c.failedJobGenerateListMutex.Unlock()
	}()

	failedURLPair := c.failedJobGenerateList.Front()
	if failedURLPair == nil {
		return nil, true
	}
	c.failedJobGenerateList.Remove(failedURLPair)

	return failedURLPair.Value.(*URLPair), false
}

// PutAFailedURLPair puts a URLPair to failedJobGenerateList
func (c *Client) PutAFailedURLPair(failedURLPair *URLPair) {
	c.failedJobGenerateListMutex.Lock()
	defer func() {
		c.failedJobGenerateListMutex.Unlock()
	}()

	if c.failedJobGenerateList != nil {
		c.failedJobGenerateList.PushBack(failedURLPair)
	}

}

// GetAFailedGenNormalURLPair get a URLPair from failedGenNormalURLPairList
func (c *Client) GetAFailedGenNormalURLPair() (*URLPair, bool) {
	c.failedGenNormalURLPairListMutex.Lock()
	defer func() {
		c.failedGenNormalURLPairListMutex.Unlock()
	}()

	failedURLPair := c.failedGenNormalURLPairList.Front()
	if failedURLPair == nil {
		return nil, true
	}
	c.failedGenNormalURLPairList.Remove(failedURLPair)

	return failedURLPair.Value.(*URLPair), false
}

// PutAFailedGenNormalURLPair puts a URLPair to failedGenNormalURLPairList
func (c *Client) PutAFailedGenNormalURLPair(failedURLPair *URLPair) {
	c.failedGenNormalURLPairListMutex.Lock()
	defer func() {
		c.failedGenNormalURLPairListMutex.Unlock()
	}()

	if c.failedGenNormalURLPairList != nil {
		c.failedGenNormalURLPairList.PushBack(failedURLPair)
	}

}

// GenJobFilterTag is hornor by TagExistOverridden policy, skip generate job if tag in target and tag digest is same
func (c *Client) GenJobFilterTag(sourceTags, targetTags []string, sourceURL, targetURL *utils.RepoURL, sourceSecurity, targetSecurity configs.Security, wg *sync.WaitGroup) {
	tagChan := make(chan string, len(sourceTags))
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, tag := range sourceTags {
			tagChan <- tag
		}
		close(tagChan)
	}()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for tag := range tagChan {
				urlPair := &URLPair{
					source: sourceURL.GetURL() + ":" + tag,
					target: targetURL.GetURL() + ":" + tag,
				}

				log.Debugf("handle tag %s", urlPair.source)
				//source tag exist in target
				if utils.IsContain(targetTags, tag) {
					if !c.config.FlagConf.Config.TagExistOverridden {
						log.Warnf("Skip push image, target image %s/%s:%s already exist, flag \"--tag-exist-overridden\" is set so skip", targetURL.GetRegistry(), targetURL.GetRepoWithNamespace(), tag)
						continue
					}
					imageSource, err := transfer.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(),
						tag, sourceSecurity.Username, sourceSecurity.Password, sourceSecurity.Insecure)
					if err != nil {
						log.Errorf("generate %s image source error: %v", sourceURL.GetURL(), err)
						c.PutAFailedGenNormalURLPair(urlPair)
						continue
					}

					imageTarget, err := transfer.NewImageTarget(targetURL.GetRegistry(), targetURL.GetRepoWithNamespace(), tag, targetSecurity.Username, targetSecurity.Password, targetSecurity.Insecure)
					if err != nil {
						log.Errorf("generate %s image target error: %v", targetURL.GetURL(), err)
						c.PutAFailedGenNormalURLPair(urlPair)
						continue
					}
					sourceDigest, err := imageSource.GetImageDigest()
					if err != nil {
						log.Errorf("Failed to get source image digest from %s/%s:%s error: %v", imageSource.GetRegistry(), imageSource.GetRepository(), tag, err)
						c.PutAFailedGenNormalURLPair(urlPair)
						continue
					}
					targetDigest, err := imageTarget.GetImageDigest()
					if err != nil {
						log.Errorf("Failed to get target image digest from %s/%s:%s error: %v", imageTarget.GetRegistry(), imageTarget.GetRepository(), tag, err)
						c.PutAFailedGenNormalURLPair(urlPair)
						continue
					}

					if sourceDigest == targetDigest {
						log.Infof("Skip push image, target image %s/%s:%s already exist and has same digest %s", imageTarget.GetRegistry(), imageTarget.GetRepository(), imageTarget.GetTag(), sourceDigest)
						continue
					}

					if targetDigest != "" {
						log.Warnf("Target image %s/%s:%s already exist, target digest %s to be override as source digest %s", imageTarget.GetRegistry(), imageTarget.GetRepository(), imageTarget.GetTag(), targetDigest, sourceDigest)
					}
				}
				log.Infof("put normal url pair %v", urlPair)
				c.PutNormalURLPair(urlPair)
			}
		}()
	}
}

// HandleCcrToTCrTags get tags from ccr api and generate urlPair by filter tag
func (c *Client) HandleCcrToTCrTags(repoChan chan string) error {
	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ccrRepo := range repoChan {
				log.Infof("ccr repo is %s", ccrRepo)
				source := fmt.Sprintf("%s%s%s", ccrapis.RegionPrefix[c.config.FlagConf.Config.CCRRegion], ".ccs.tencentyun.com/", ccrRepo)
				target := c.config.FlagConf.Config.TCRName + ".tencentcloudcr.com/" + ccrRepo
				urlPair := &URLPair{
					source: source,
					target: target,
				}

				err := c.GenCcrtoTcrTagURLPair(source, target, &wg)
				if err != nil {
					c.PutAFailedGenNormalURLPair(urlPair)
					log.Errorf("Handle repoChan tags failed error, %s", err)
				}
			}
		}()
	}
	wg.Wait()
	return nil
}

// SetURLPairFinished set finished flag
func (c *Client) SetURLPairFinished() {
	c.urlPairFinishedMutex.Lock()
	defer c.urlPairFinishedMutex.Unlock()
	c.urlPairFinished = true
}

// IsURLPairFinished locked and return c.urlPairFinished
func (c *Client) IsURLPairFinished() bool {
	c.urlPairFinishedMutex.Lock()
	defer c.urlPairFinishedMutex.Unlock()
	return c.urlPairFinished
}

// GenTagURLPair is generate normal image url that containt tag
func (c *Client) GenTagURLPair(source string, target string, wg *sync.WaitGroup) error {
	if source == "" {
		return fmt.Errorf("source url should not be empty")
	}

	sourceURL, err := utils.NewRepoURL(source)
	if err != nil {
		return fmt.Errorf("url %s format error: %v", source, err)
	}

	// if dest is not specific, use default registry and namespace
	if target == "" {
		if c.config.FlagConf.Config.DefaultRegistry != "" && c.config.FlagConf.Config.DefaultNamespace != "" {
			target = c.config.FlagConf.Config.DefaultRegistry + "/" +
				c.config.FlagConf.Config.DefaultNamespace + "/" + sourceURL.GetRepoWithTag()
		} else {
			return fmt.Errorf("the default registry and namespace should not be nil if you want to use them")
		}
	}

	targetURL, err := utils.NewRepoURL(target)
	if err != nil {
		return fmt.Errorf("url %s format error: %v", target, err)
	}

	var imageSource *transfer.ImageSource
	var imageTarget *transfer.ImageTarget

	sourceSecurity, exist := c.config.GetSecuritySpecific(sourceURL.GetRegistry(), sourceURL.GetNamespace())
	if exist {
		log.Infof("Find auth information for %v, username: %v", sourceURL.GetURL(), sourceSecurity.Username)
	} else {
		log.Infof("Cannot find auth information for %v, pull actions will be anonymous", sourceURL.GetURL())
	}

	targetSecurity, exist := c.config.GetSecuritySpecific(targetURL.GetRegistry(), targetURL.GetNamespace())
	if exist {
		log.Infof("Find auth information for %v, username: %v", targetURL.GetURL(), targetSecurity.Username)

	} else {
		log.Infof("Cannot find auth information for %v, push actions will be anonymous", targetURL.GetURL())
	}

	// multi-tags config
	tags := sourceURL.GetTag()
	if moreTag := strings.Split(tags, ","); len(moreTag) > 1 {
		if targetURL.GetTag() != "" && targetURL.GetTag() != sourceURL.GetTag() {
			return fmt.Errorf("multi-tags source should not correspond to a target with tag: %s:%s",
				sourceURL.GetURL(), targetURL.GetURL())
		}
		log.Debugf("source %s tags is %s", sourceURL.GetURL(), moreTag)
		imageTarget, err = transfer.NewImageTarget(targetURL.GetRegistry(), targetURL.GetRepoWithNamespace(), targetURL.GetTag(), targetSecurity.Username, targetSecurity.Password, targetSecurity.Insecure)
		if err != nil {
			return fmt.Errorf("generate %s image target error: %v", sourceURL.GetURL(), err)
		}
		targetTags, err := imageTarget.GetTargetRepoTags()
		log.Debugf("target %s tags is %s", targetURL.GetURL(), targetTags)
		if err != nil {
			return fmt.Errorf("get tags failed from %s error: %v", targetURL.GetURL(), err)
		}
		c.GenJobFilterTag(moreTag, targetTags, sourceURL, targetURL, sourceSecurity, targetSecurity, wg)
		return nil
	}

	imageSource, err = transfer.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(), sourceURL.GetTag(), sourceSecurity.Username, sourceSecurity.Password, sourceSecurity.Insecure)
	if err != nil {
		return fmt.Errorf("generate %s image source error: %v", sourceURL.GetURL(), err)
	}

	imageTarget, err = transfer.NewImageTarget(targetURL.GetRegistry(), targetURL.GetRepoWithNamespace(), targetURL.GetTag(), targetSecurity.Username, targetSecurity.Password, targetSecurity.Insecure)
	if err != nil {
		return fmt.Errorf("generate %s image target error: %v", sourceURL.GetURL(), err)
	}

	// if tag is not specific, return tags
	if sourceURL.GetTag() == "" {
		if targetURL.GetTag() != "" {
			return fmt.Errorf("not allow source tag empty and target tag not empty, both side of the config: %s:%s",
				sourceURL.GetURL(), targetURL.GetURL())
		}

		// get all tags of this source repo
		sourceTags, err := imageSource.GetSourceRepoTags()
		log.Debugf("source %s tags is %s", sourceURL.GetURL(), sourceTags)
		if err != nil {
			return fmt.Errorf("get tags failed from %s error: %v", sourceURL.GetURL(), err)
		}

		targetTags, err := imageTarget.GetTargetRepoTags()
		log.Debugf("target %s tags is %s", targetURL.GetURL(), targetTags)
		if err != nil {
			return fmt.Errorf("get tags failed from %s error: %v", targetURL.GetURL(), err)
		}

		c.GenJobFilterTag(sourceTags, targetTags, sourceURL, targetURL, sourceSecurity, targetSecurity, wg)
		return nil
	}

	// if source tag is set but without destinate tag, use the same tag as source
	destTag := targetURL.GetTag()
	if destTag == "" {
		destTag = sourceURL.GetTag()
	}

	imageTarget, err = transfer.NewImageTarget(targetURL.GetRegistry(), targetURL.GetRepoWithNamespace(),
		destTag, targetSecurity.Username, targetSecurity.Password, targetSecurity.Insecure)
	if err != nil {
		return fmt.Errorf("generate %s image target error: %v", sourceURL.GetURL(), err)
	}

	sourceDigest, err := imageSource.GetImageDigest()
	if err != nil {
		log.Errorf("Failed to get source image digest from %s/%s:%s error: %v", imageSource.GetRegistry(), imageSource.GetRepository(), sourceURL.GetTag(), err)
		return err
	}
	targetDigest, err := imageTarget.GetImageDigest()
	if err != nil {
		log.Errorf("Failed to get target image digest from %s/%s:%s error: %v", imageTarget.GetRegistry(), imageTarget.GetRepository(), destTag, err)
		return err
	}

	if sourceDigest == targetDigest {
		log.Infof("Skip push image, target image %s/%s:%s already exist and has same digest %s", imageTarget.GetRegistry(), imageTarget.GetRepository(), imageTarget.GetTag(), sourceDigest)
		return nil
	}

	c.PutNormalURLPair(&URLPair{
		source: source,
		target: target,
	})
	log.Infof("put normal url pair source: %s, target: %s", source, target)
	return nil
}

// HandleURLPair put urlPair to normalURLPair
func (c *Client) HandleURLPair() {
	routineNum := c.config.FlagConf.Config.RoutineNums
	wg := sync.WaitGroup{}

	for i := 0; i < routineNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer log.Infof("exit HandleURLPair main loop")
			for {
				urlPair, empty := c.GetURLPair()
				// no more job to generate
				if empty {
					log.Infof("HandleURLPair is empty")
					break
				}
				err := c.GenTagURLPair(urlPair.source, urlPair.target, &wg)
				if err != nil {
					log.Errorf("Generate tag urlPair %s to %s error: %v", urlPair.source, urlPair.target, err)
					// put to failedGenNormalURLPair
					c.PutAFailedGenNormalURLPair(urlPair)
				}
			}
		}()
	}
	wg.Wait()
}

// CcrtoTcrGenTagRetry put urlPair to normalURLPair if job is CcrtoTcr
func (c *Client) CcrtoTcrGenTagRetry() {
	routineNum := 5
	wg := sync.WaitGroup{}

	for i := 0; i < routineNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer log.Debugf("exit CcrtoTcrGenTagRetry main loop")
			for {
				urlPair, empty := c.GetURLPair()
				// no more job to generate
				if empty {
					log.Debugf("CcrtoTcrGenTagRetry url pair is empty")
					break
				}
				err := c.GenCcrtoTcrTagURLPair(urlPair.source, urlPair.target, &wg)
				if err != nil {
					log.Errorf("Generate tag urlPair %s to %s error: %v", urlPair.source, urlPair.target, err)
					// put to failedGenNormalURLPair
					c.PutAFailedGenNormalURLPair(urlPair)
				}
			}
		}()
	}
	wg.Wait()
}

// GenCcrtoTcrTagURLPair is generate normal url pair
func (c *Client) GenCcrtoTcrTagURLPair(source string, target string, wg *sync.WaitGroup) error {
	urlPair := &URLPair{
		source: source,
		target: target,
	}

	sourceURL, err := utils.NewRepoURL(source)
	if err != nil {
		return fmt.Errorf("url %s format error: %v", source, err)
	}

	targetURL, err := utils.NewRepoURL(target)
	if err != nil {
		return fmt.Errorf("url %s format error: %v", target, err)
	}

	sourceSecurity, exist := c.config.GetSecuritySpecific(sourceURL.GetRegistry(), sourceURL.GetNamespace())
	if exist {
		log.Infof("Find auth information for %v, username: %v", sourceURL.GetURL(), sourceSecurity.Username)
	} else {
		log.Infof("Cannot find auth information for %v, pull actions will be anonymous", sourceURL.GetURL())
	}

	targetSecurity, exist := c.config.GetSecuritySpecific(targetURL.GetRegistry(), targetURL.GetNamespace())
	if exist {
		log.Infof("Find auth information for %v, username: %v", targetURL.GetURL(), targetSecurity.Username)

	} else {
		log.Infof("Cannot find auth information for %v, push actions will be anonymous", targetURL.GetURL())
	}

	if sourceURL.GetTag() == "" {
		ccrClient := ccrapis.NewCCRAPIClient()

		ccrSecretID, ccrSecretKey, err := ccrapis.GetCcrSecret(c.config.Secret)
		if err != nil {
			log.Errorf("GetCcrSecret error: %s", err)
			return err
		}

		sourceTags, err := ccrClient.GetRepoTags(ccrSecretID, ccrSecretKey, c.config.FlagConf.Config.CCRRegion, sourceURL.GetRepoWithNamespace())
		log.Debugf("ccr target %s tags is %s", sourceURL.GetOriginURL(), sourceTags)
		if err != nil {
			log.Errorf("Failed get ccr repo %s tags, error: %s", sourceURL.GetRepoWithNamespace(), err)
			return fmt.Errorf("failed get ccr repo %s tags, error: %s", sourceURL.GetRepoWithNamespace(), err)
		}

		imageTarget, err := transfer.NewImageTarget(targetURL.GetRegistry(), targetURL.GetRepoWithNamespace(), targetURL.GetTag(), targetSecurity.Username, targetSecurity.Password, targetSecurity.Insecure)
		if err != nil {
			return fmt.Errorf("generate %s image target error: %v", sourceURL.GetURL(), err)
		}

		targetTags, err := imageTarget.GetTargetRepoTags()
		log.Debugf("target %s tags is %s", targetURL.GetURL(), targetTags)
		if err != nil {
			return fmt.Errorf("get tags failed from %s error: %v", targetURL.GetURL(), err)
		}

		log.Debugf("GenCcrtoTcrTagURLPair call GenJobFilterTag")
		c.GenJobFilterTag(sourceTags, targetTags, sourceURL, targetURL, sourceSecurity, targetSecurity, wg)
		return nil
	}

	// already contain tag

	imageSource, err := transfer.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(), sourceURL.GetTag(), sourceSecurity.Username, sourceSecurity.Password, sourceSecurity.Insecure)
	if err != nil {
		return fmt.Errorf("generate %s image source error: %v", sourceURL.GetURL(), err)
	}

	imageTarget, err := transfer.NewImageTarget(targetURL.GetRegistry(), targetURL.GetRepoWithNamespace(), targetURL.GetTag(), targetSecurity.Username, targetSecurity.Password, targetSecurity.Insecure)
	if err != nil {
		return fmt.Errorf("generate %s image target error: %v", sourceURL.GetURL(), err)
	}

	sourceDigest, err := imageSource.GetImageDigest()
	if err != nil {
		log.Errorf("Failed to get source image digest from %s/%s:%s error: %v", imageSource.GetRegistry(), imageSource.GetRepository(), sourceURL.GetTag(), err)
		return err
	}
	targetDigest, err := imageTarget.GetImageDigest()
	if err != nil {
		log.Errorf("Failed to get target image digest from %s/%s:%s error: %v", imageTarget.GetRegistry(), imageTarget.GetRepository(), targetURL.GetTag(), err)
		return err
	}

	if sourceDigest == targetDigest {
		log.Infof("Skip push image, target image %s/%s:%s already exist and has same digest %s", imageTarget.GetRegistry(), imageTarget.GetRepository(), imageTarget.GetTag(), sourceDigest)
		return nil
	}

	c.PutNormalURLPair(urlPair)
	log.Infof("put normal url pair source: %s, target: %s", source, target)
	return nil
}
