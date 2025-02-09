/*
Copyright 2014 The Kubernetes Authors.

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

package validate

import (
	"context"

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	netutils "k8s.io/utils/net"
)

// IPSloppy verifies that the specified value is a valid IP address,
// but allows leading zeros on each octet value.
// This should not be used for new APIs.
//
// `T` can be either ~string or ~*string. Requires Go 1.21+ for union constraints.
func IPSloppy[T ~string | ~*string](
	ctx context.Context,
	op operation.Operation,
	fldPath *field.Path,
	value, _ *T,
) field.ErrorList {
	// If value is nil, skip
	if value == nil {
		return nil
	}

	// Switch on the underlying type
	switch v := any(*value).(type) {
	case string:
		return validateIPSloppyString(ctx, op, fldPath, v)
	case *string:
		if v == nil {
			return nil
		}
		return validateIPSloppyString(ctx, op, fldPath, *v)
	default:
		return field.ErrorList{
			field.Invalid(fldPath, *value, "expected type ~string or ~*string for IP sloppy validation"),
		}
	}
}

// validateIPSloppyString is a helper that runs the actual sloppy IP parse on a raw string.
func validateIPSloppyString(
	_ context.Context,
	_ operation.Operation,
	fldPath *field.Path,
	s string,
) field.ErrorList {
	var errs field.ErrorList
	ip := netutils.ParseIPSloppy(s)
	if ip == nil {
		errs = append(errs, field.Invalid(
			fldPath, s,
			"must be a valid IP address (e.g. 10.9.8.7 or 2001:db8::ffff)",
		).WithOrigin("format=ip-sloppy"))
	}
	return errs
}
