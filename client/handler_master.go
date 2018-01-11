package client

import (
	"fmt"
	"log"
	"github.com/OpenSLX/bwlp-go-client/bwlp"
)

func (handler *SessionHandler) Login(username string, password string) error {
	// perform login
	session, err := handler.GetMasterClient().LocalAccountLogin(username, password)
	if err != nil {
		log.Printf("## Authentication failed: %s\n", err)
		return err
	}
	// store session data for later use
	handler.SessionData = session
	log.Printf("** Registered satellites: %s", handler.SessionData.Satellites)
	// TODO handle/auto sat selection
	return nil
}

func (handler* SessionHandler) GetPublicImageList(page int32) ([]*bwlp.ImageSummaryRead, error) {
	imageList, err := handler.GetMasterClient().GetPublicImages(handler.SessionData.SessionId, page)
	if err != nil {
		log.Printf("Error requesting public image list from masterserver.\n")
		return nil, err
	}
	return imageList, nil
}

func (handler* SessionHandler) GetPublicImageDetails(imageBaseID string) (*bwlp.ImageDetailsRead, error) {
	// request image details
	imageDetails, err := handler.GetMasterClient().GetImageDetails(handler.SessionData.SessionId, bwlp.UUID(imageBaseID))
	if err != nil {
		log.Printf("Failed to retrieve image details for '%s': %s\n", imageBaseID, err)
		return nil, err
	}
	return imageDetails, nil
}

func (handler* SessionHandler) GetLatestVersion(imageBaseID string) (*bwlp.ImageVersionDetails, error) {
	// get image details in bwlp's system to retrieve the last version
	imageDetails, err := handler.GetPublicImageDetails(imageBaseID)
	if err != nil {
		return nil, err
	}
	// just to make sure the LatestVersionId actually exists...
	for _, version := range imageDetails.Versions {
		if version.VersionId == imageDetails.LatestVersionId {
			return version, nil
		}
	}
	return nil, fmt.Errorf("Could not find latest version for '%s'\n", imageBaseID)
}

func (handler* SessionHandler) GetPublicImageData(imageBaseID string) (*Transfer, error) {
	// get image details in bwlp's system to retrieve the last version
	imageDetails, err := handler.GetPublicImageDetails(imageBaseID)
	if err != nil {
		return nil, err
	}
	// just to make sure the LatestVersionId actually exists...
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
	ti, err := handler.GetMasterClient().DownloadImage(handler.SessionData.AuthToken, imageVersion.VersionId)
	if err != nil {
		log.Printf("Error requesting download of image version '%s': %s\n", imageVersion.VersionId, err)
		return nil, err
	}
	// in case of vmware images, the vmx is contained within the TransferInformation
	// TODO handle description files using specifics most likely
	return NewTransfer(false, handler.masterEndpoint.Hostname, ti, imageVersion.FileSize), nil
}
