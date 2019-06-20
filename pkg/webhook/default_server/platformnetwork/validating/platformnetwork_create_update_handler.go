/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package validating

import (
	"context"
	"github.com/wind-river/titanium-deployment-manager/pkg/common"
	"net/http"

	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Webhook response reasons
const AllowedReason string = "allowed to be admitted"

const (
	// Maximum IP address prefix lengths
	MaxIPv4PrefixLength int = 32
	MaxIPv6PrefixLength int = 128
)

func init() {
	webhookName := "validating-create-update-platformnetwork"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &PlatformNetworkCreateUpdateHandler{})
}

// PlatformNetworkCreateUpdateHandler handles PlatformNetwork
type PlatformNetworkCreateUpdateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// client  client.client

	// Decoder decodes objects
	Decoder types.Decoder
}

// Determines if a string is a valid IP address
func IsIPAddress(value string) bool {
	return common.IsIPv4(value) || common.IsIPv6(value)
}

// Determines if the a prefix length agrees with the address family of the specified address
func IsValidPrefix(address string, prefix int) bool {
	if common.IsIPv4(address) {
		if prefix <= MaxIPv4PrefixLength {
			return true
		}
	} else if common.IsIPv6(address) {
		if prefix <= MaxIPv6PrefixLength {
			return true
		}
	}
	return false
}

// Validates that all address specifications within the network are of the same address family.
func (h *PlatformNetworkCreateUpdateHandler) validateAddressFamilies(obj *starlingxv1beta1.PlatformNetwork) (bool, string, error) {
	if IsIPAddress(obj.Spec.Subnet) != true {
		return false, "expecting a valid IPv4 or IPv6 address in subnet", nil
	}

	if IsValidPrefix(obj.Spec.Subnet, obj.Spec.Prefix) != true {
		return false, "prefix value must correspond to the subnet address family", nil
	}

	for _, r := range obj.Spec.Allocation.Ranges {
		if IsIPAddress(r.Start) != true || IsIPAddress(r.End) != true {
			return false, "start and end addresses must be valid IP addresses", nil
		}

		if common.IsIPv4(r.Start) != common.IsIPv4(r.End) {
			return false, "start and end addresses must be of the same address family", nil
		}

		if common.IsIPv4(r.Start) != common.IsIPv4(obj.Spec.Subnet) {
			return false, "allocation range address must be of the same family as the network subnet.", nil
		}
	}

	return true, AllowedReason, nil
}

// Validates an incoming resource update/create request.  The intent of this validation is to perform only the
// minimum amount of validation which should normally be done by the CRD validation schema, but until kubebuilder
// supports the necessary validation annotations we need to do this in a webhook.  All other validation is left
// to the system API and any errors generated by that API will be reported in the resource status and events.
func (h *PlatformNetworkCreateUpdateHandler) validatingPlatformNetworkFn(ctx context.Context, obj *starlingxv1beta1.PlatformNetwork) (bool, string, error) {
	allowed, reason, err := h.validateAddressFamilies(obj)
	if !allowed || err != nil {
		return allowed, reason, err
	}

	return allowed, reason, err
}

var _ admission.Handler = &PlatformNetworkCreateUpdateHandler{}

// Handle handles admission requests.
func (h *PlatformNetworkCreateUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &starlingxv1beta1.PlatformNetwork{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.validatingPlatformNetworkFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

//var _ inject.client = &PlatformNetworkCreateUpdateHandler{}
//
//// InjectClient injects the client into the PlatformNetworkCreateUpdateHandler
//func (h *PlatformNetworkCreateUpdateHandler) InjectClient(c client.client) error {
//	h.client = c
//	return nil
//}

var _ inject.Decoder = &PlatformNetworkCreateUpdateHandler{}

// InjectDecoder injects the decoder into the PlatformNetworkCreateUpdateHandler
func (h *PlatformNetworkCreateUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}