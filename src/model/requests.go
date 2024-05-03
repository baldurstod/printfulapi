package model

import (
	printfulAPIModel "github.com/baldurstod/printful-api-model"
)

type CalculateShippingRates struct {
	Recipient printfulAPIModel.AddressInfo `mapstructure:"recipient"`
	Items     []printfulAPIModel.ItemInfo  `mapstructure:"items"`
	Currency  string                       `mapstructure:"currency"`
	Locale    string                       `mapstructure:"locale"`
}
