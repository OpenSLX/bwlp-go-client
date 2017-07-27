package client

import (
	"fmt"
	"log"
	"github.com/OpenSLX/bwlp-go-client/bwlp"
)

type SessionHandler struct {
	Client *bwlp.MasterServerClient
	SessionData *bwlp.ClientSessionData
	Endpoint *MasterServerEndpoint
}

func NewSessionHandler(endpoint *MasterServerEndpoint) (*SessionHandler, error) {
	newHandler := SessionHandler{
		Client: nil,
		SessionData: nil,
		Endpoint: nil,
	}
	if err := newHandler.initClient(endpoint); err != nil {
		return nil, err // errors are handled by initClient
	} else {
		return &newHandler, nil
	}
}

func (handler *SessionHandler) Login(username string, password string) error {
	if handler.Client == nil {
		return fmt.Errorf("bwLehrpool client not initialized.")
	}
	// perform login
	session, err := handler.Client.LocalAccountLogin(username, password)
	if err != nil {
		log.Printf("## Authentication failed: %s\n", err)
		return err
	}
	// store session data for later use
	handler.SessionData = session
	return nil
}

func (handler* SessionHandler) GetImageDetails(imageBaseID string) (*bwlp.ImageDetailsRead, error) {
	if handler.Client == nil || handler.SessionData == nil {
		return nil, fmt.Errorf("bwLehrpool client not initialized or not authenticated.")
	}
	// request image details
	imageDetails, err := handler.Client.GetImageDetails(handler.SessionData.SessionId, bwlp.UUID(imageBaseID))
	if err != nil {
		log.Printf("Failed to retrieve image details for '%s': %s\n", imageBaseID, err)
		return nil, err
	}
	return imageDetails, nil
}

func (handler* SessionHandler) GetImageData(imageBaseID string) (*Downloader, error) {
	// trigger download
	imageDetails, err := handler.GetImageDetails(imageBaseID)
	if err != nil {
		return nil, err
	}
	// TODO handle versions, for now just use latest one
  var imageVersion *bwlp.ImageVersionDetails = nil
  for _, version := range imageDetails.Versions {
    if version.VersionId == imageDetails.LatestVersionId {
      imageVersion = version
    }
  }
  if imageVersion == nil {
    return nil, fmt.Errorf("Latest version not found in image version list, this should not happen :)") 
  }
  // Request download of that version
  ti, err := handler.Client.DownloadImage(handler.SessionData.AuthToken, imageVersion.VersionId)
  if err != nil {
    log.Printf("Error requesting download of image version '%s': %s\n", imageVersion.VersionId, err)
    return nil, err
  }
	// in case of vmware images, the vmx is contained within the TransferInformation
	// TODO handle description files using specifics most likely
	downloader := NewDownloader(handler.Endpoint.Hostname, ti, imageVersion)
	return downloader, nil
}

func (handler *SessionHandler) initClient(endpoint *MasterServerEndpoint) error {
	// already initialised?
	if handler.Client != nil {
		log.Printf("Masterserver client already initialized.\n")
		return nil
	}
	// set environment's endpoint
	if err := SetEndpoint(endpoint); err != nil {
		log.Printf("Error setting endpoint during masterserver client initialisation: %s\n", err)
		return err
	}
	handler.Endpoint = endpoint
	// get main masterserver client instance
	client := GetInstance()

	// verify that connection is established,
	_, err := client.Ping()
	if err != nil {
		log.Printf("Error pinging masterserver: %s\n", err)
		return err
	}
	handler.Client = client
	return nil
}

