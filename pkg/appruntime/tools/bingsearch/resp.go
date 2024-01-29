/*
Copyright 2024 KubeAGI.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bingsearch

import "time"

type RespData struct {
	Type            string          `json:"_type,omitempty"`
	QueryContext    QueryContext    `json:"queryContext,omitempty"`
	WebPages        WebPages        `json:"webPages,omitempty"`
	Entities        Entities        `json:"entities,omitempty"`
	Videos          Videos          `json:"videos,omitempty"`
	News            News            `json:"news,omitempty"`
	RankingResponse RankingResponse `json:"rankingResponse,omitempty"`
	ErrorResp       *ErrorResp      `json:"error,omitempty"`
}

type QueryContext struct {
	OriginalQuery string `json:"originalQuery"`
}

type WebPages struct {
	WebSearchURL          string          `json:"webSearchUrl"`
	TotalEstimatedMatches int             `json:"totalEstimatedMatches"`
	Value                 []WebPagesValue `json:"value"`
	SomeResultsRemoved    bool            `json:"someResultsRemoved"`
}

type WebPagesValue struct {
	ID                       string             `json:"id"`
	Name                     string             `json:"name"`
	URL                      string             `json:"url"`
	DatePublished            string             `json:"datePublished,omitempty"`
	DatePublishedDisplayText string             `json:"datePublishedDisplayText,omitempty"`
	IsFamilyFriendly         bool               `json:"isFamilyFriendly"`
	DisplayURL               string             `json:"displayUrl"`
	Snippet                  string             `json:"snippet"`
	DeepLinks                []DeepLink         `json:"deepLinks,omitempty"`
	DateLastCrawled          time.Time          `json:"dateLastCrawled"`
	CachedPageURL            string             `json:"cachedPageUrl,omitempty"`
	Language                 string             `json:"language"`
	IsNavigational           bool               `json:"isNavigational"`
	ThumbnailURL             string             `json:"thumbnailUrl,omitempty"`
	PrimaryImageOfPage       PrimaryImageOfPage `json:"primaryImageOfPage,omitempty"`
	SearchTags               []SearchTag        `json:"searchTags,omitempty"`
}

type DeepLink struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type PrimaryImageOfPage struct {
	ThumbnailURL string `json:"thumbnailUrl"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ImageID      string `json:"imageId"`
}

type SearchTag struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Entities struct {
	Value []EntitiesValue `json:"value"`
}

type EntitiesValue struct {
	ID                     string                 `json:"id"`
	EntityPresentationInfo EntityPresentationInfo `json:"entityPresentationInfo"`
	BingID                 string                 `json:"bingId"`
}

type EntityPresentationInfo struct {
	EntityScenario string `json:"entityScenario"`
}

type RankingResponse struct {
	Mainline RankingResponseMainline `json:"mainline"`
	Sidebar  RankingResponseSidebar  `json:"sidebar"`
}

type RankingResponseMainline struct {
	Items []RankingResponseItem `json:"items"`
}

type RankingResponseSidebar struct {
	Items []RankingResponseItem `json:"items"`
}

type RankingResponseItem struct {
	AnswerType  string                   `json:"answerType"`
	ResultIndex int                      `json:"resultIndex"`
	Value       RankingResponseItemValue `json:"value"`
}

type RankingResponseItemValue struct {
	ID string `json:"id"`
}

type ErrorResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Videos struct {
	ID               string       `json:"id"`
	ReadLink         string       `json:"readLink"`
	WebSearchURL     string       `json:"webSearchUrl"`
	IsFamilyFriendly bool         `json:"isFamilyFriendly"`
	Value            []VideoValue `json:"value"`
	Scenario         string       `json:"scenario"`
}

type VideoValue struct {
	WebSearchURL       string      `json:"webSearchUrl"`
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	ThumbnailURL       string      `json:"thumbnailUrl"`
	DatePublished      string      `json:"datePublished"`
	Publisher          []Publisher `json:"publisher"`
	Creator            Creator     `json:"creator,omitempty"`
	ContentURL         string      `json:"contentUrl"`
	HostPageURL        string      `json:"hostPageUrl"`
	EncodingFormat     string      `json:"encodingFormat"`
	HostPageDisplayURL string      `json:"hostPageDisplayUrl"`
	Width              int         `json:"width"`
	Height             int         `json:"height"`
	Duration           string      `json:"duration,omitempty"`
	MotionThumbnailURL string      `json:"motionThumbnailUrl,omitempty"`
	EmbedHTML          string      `json:"embedHtml"`
	AllowHTTPSEmbed    bool        `json:"allowHttpsEmbed"`
	ViewCount          int         `json:"viewCount"`
	Thumbnail          Thumbnail   `json:"thumbnail"`
	AllowMobileEmbed   bool        `json:"allowMobileEmbed"`
	IsSuperfresh       bool        `json:"isSuperfresh"`
}

type Thumbnail struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	ContentURL string `json:"contentUrl,omitempty"`
}

type Publisher struct {
	Name string `json:"name"`
}

type Creator struct {
	Name string `json:"name"`
}

type News struct {
	ID         string      `json:"id"`
	ReadLink   string      `json:"readLink"`
	NewsValues []NewsValue `json:"value"`
}

type NewsValue struct {
	ContractualRules []ContractualRules `json:"contractualRules"`
	Name             string             `json:"name"`
	URL              string             `json:"url"`
	Description      string             `json:"description"`
	Provider         []Provider         `json:"provider"`
	DatePublished    time.Time          `json:"datePublished"`
	Category         string             `json:"category"`
	Image            Image              `json:"image,omitempty"`
}

type ContractualRules struct {
	Type string `json:"_type"`
	Text string `json:"text"`
}

type Provider struct {
	Type  string `json:"_type"`
	Name  string `json:"name"`
	Image Image  `json:"image"`
}

type Image struct {
	ContentURL string    `json:"contentUrl"`
	Thumbnail  Thumbnail `json:"thumbnail"`
}
