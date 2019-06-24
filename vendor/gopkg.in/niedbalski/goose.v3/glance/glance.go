// goose/glance - Go package to interact with OpenStack Image Service (Glance) API.
// For V2 functions see http://developer.openstack.org/api-ref/image/v2/index.html
// For all other functions see http://developer.openstack.org/api-ref/compute/#list-images

package glance

import (
	"fmt"
	"os"
	"net/http"
	"time"
	"gopkg.in/niedbalski/goose.v3/client"
	"gopkg.in/niedbalski/goose.v3/errors"
	goosehttp "gopkg.in/niedbalski/goose.v3/http"
)

// API URL parts.
const (
	apiImages       = "/images"
	apiImagesDetail = "/images/detail"
)

// Client provides a means to access the OpenStack Image Service.
type Client struct {
	client client.Client
}

// New creates a new Client.
func New(client client.Client) *Client {
	return &Client{client}
}

// Link describes a link to an image in OpenStack.
type Link struct {
	Href string
	Rel  string
	Type string
}

// Image describes an OpenStack image.
type Image struct {
	Id    string
	Name  string
	Links []Link
}

type ImageOpts struct {
	ContainerFormat string `json:"container_format"`
	DiskFormat      string `json:"disk_format"`
	Name            string `json:"name"`
	Protected       bool `json:"protected"`
	Visibility      string `json:"visibility"`
}

type CreateImageResponse struct {
	Status          string        `json:"status"`
	Name            string        `json:"name"`
	Tags            []interface{} `json:"tags"`
	ContainerFormat string        `json:"container_format"`
	CreatedAt       time.Time     `json:"created_at"`
	Size            interface{}   `json:"size"`
	DiskFormat      string        `json:"disk_format"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Visibility      string        `json:"visibility"`
	Locations       []interface{} `json:"locations"`
	Self            string        `json:"self"`
	MinDisk         int           `json:"min_disk"`
	Protected       bool          `json:"protected"`
	ID              string        `json:"id"`
	File            string        `json:"file"`
	Checksum        interface{}   `json:"checksum"`
	OsHashAlgo      interface{}   `json:"os_hash_algo"`
	OsHashValue     interface{}   `json:"os_hash_value"`
	OsHidden        bool          `json:"os_hidden"`
	Owner           string        `json:"owner"`
	VirtualSize     interface{}   `json:"virtual_size"`
	MinRAM          int           `json:"min_ram"`
	Schema          string        `json:"schema"`
}

func (c *Client) CreateImageFromFile(filePath string, opts ImageOpts) (*CreateImageResponse, error){
	var resp CreateImageResponse

	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Newf(err, "cannot open image filename %#v", filePath)
	}

	defer file.Close()

	requestData := goosehttp.RequestData{ReqValue: opts, RespValue: &resp, ExpectedStatus: []int{http.StatusCreated, http.StatusAccepted}}
	err = c.client.SendRequest(client.POST, "image", "v2", apiImages, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to create image with %#v", opts)
	}

	contentTypeHeader := http.Header{}
	contentTypeHeader.Set("content-type", "application/octet-stream")

	err = c.client.SendRequest(client.PUT, "image", "v2", fmt.Sprintf("%s/%s/file", apiImages, resp.ID),
		&goosehttp.RequestData{ReqHeaders: contentTypeHeader, ReqReader: file,
		ExpectedStatus: []int{http.StatusCreated, http.StatusAccepted, 204}})

	if err != nil {
		return nil, errors.Newf(err, "failed to create image with %#v, #v", opts, file)
	}

	return &resp, nil
}

// ListImages lists IDs, names, and links for available images.
func (c *Client) ListImages() ([]Image, error) {
	var resp struct {
		Images []Image
	}
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.GET, "compute", "v2", apiImages, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of images")
	}
	return resp.Images, nil
}

// ImageMetadata describes metadata of an image
type ImageMetadata struct {
	Architecture string
	State        string      `json:"image_state"`
	Location     string      `json:"image_location"`
	KernelId     interface{} `json:"kernel_id"`
	ProjectId    interface{} `json:"project_id"`
	RAMDiskId    interface{} `json:"ramdisk_id"`
	OwnerId      interface{} `json:"owner_id"`
}

// ImageDetail describes extended information about an image.
type ImageDetail struct {
	Id          string
	Name        string
	Created     string
	Updated     string
	Progress    int
	Status      string
	MinimumRAM  int `json:"minRam"`
	MinimumDisk int `json:"minDisk"`
	Links       []Link
	Metadata    ImageMetadata
}

// ListImageDetails lists all details for available images.
func (c *Client) ListImagesDetail() ([]ImageDetail, error) {
	var resp struct {
		Images []ImageDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", apiImagesDetail, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of image details")
	}
	return resp.Images, nil
}

// GetImageDetail lists details of the specified image.
func (c *Client) GetImageDetail(imageId string) (*ImageDetail, error) {
	var resp struct {
		Image ImageDetail
	}
	url := fmt.Sprintf("%s/%s", apiImages, imageId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get details of imageId: %s", imageId)
	}
	return &resp.Image, nil
}

// ImageDetailV2 describes extended information about an image for API v2.0.
type ImageDetailV2 struct {
	Architecture    string `json:",omitempty"`
	Checksum        string
	ContainerFormat string `json:"container_format"`
	CreatedAt       string `json:"created_at"`
	DirectUrl       string
	DiskFormat      string `json:"disk_format"`
	File            string
	Id              string
	HwVifModel      string      `json:"hw_vif_model,omitempty"`
	HwDiskModel     string      `json:"hw_disk_bus,omitempty"`
	Locations       interface{} `json:",omitempty"`
	MinimumDisk     int         `json:"min_disk"`
	MinimumRAM      int         `json:"min_ram"`
	Name            string
	Owner           string
	OsType          string `json:"os_type"`
	Protected       bool
	Schema          string
	Self            string
	Size            int
	Status          string
	Tags            []string
	UpdatedAt       string `json:"updated_at"`
	VirtualSize     string `json:"virtual_size"`
	Visibility      string
}

// ListImagesV2 lists all details for available image, uses API v2.0.
func (c *Client) ListImagesV2() ([]ImageDetailV2, error) {
	var resp struct {
		Images []ImageDetailV2
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "image", "v2", apiImages, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of image details (v2)")
	}
	return resp.Images, nil
}

// GetImageDetailV2 lists details of the specified image, uses API v2.0
func (c *Client) GetImageDetailV2(imageId string) (*ImageDetailV2, error) {
	var resp ImageDetailV2
	url := fmt.Sprintf("%s/%s", apiImages, imageId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "image", "v2", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get details of imageId (v2): %s", imageId)
	}
	return &resp, nil
}

