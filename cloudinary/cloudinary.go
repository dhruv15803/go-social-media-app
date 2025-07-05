package cloudinary

import "github.com/cloudinary/cloudinary-go/v2"

func LoadCloudinaryInstance() (*cloudinary.Cloudinary, error) {

	cld, err := cloudinary.New()
	if err != nil {
		return nil, err
	}

	cld.Config.URL.Secure = true

	return cld, nil
}
