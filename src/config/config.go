package config

type Config struct {
	HTTP      HTTP `json:"http"`
	Databases struct {
		Printful Database `json:"printful"`
		Images   Database `json:"images"`
	} `json:"databases"`
	Printful Printful `json:"printful"`
}

type HTTP struct {
	Port          int    `json:"port"`
	HttpsKeyFile  string `json:"https_key_file"`
	HttpsCertFile string `json:"https_cert_file"`
}

type Database struct {
	ConnectURI string `json:"connect_uri"`
	DBName     string `json:"db_name"`
	BucketName string `json:"bucket_name"`
}

type Printful struct {
	Endpoint string `json:"endpoint"`
}
