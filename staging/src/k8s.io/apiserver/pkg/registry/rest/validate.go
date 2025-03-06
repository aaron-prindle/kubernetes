/*
Copyright 2025 The Kubernetes Authors.

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

package rest

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	fldtest "k8s.io/apimachinery/pkg/util/validation/field/testing"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	validationmetrics "k8s.io/component-base/metrics/prometheus/validation"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/features"
)

// ValidateDeclaratively validates obj against declarative validation tags
// defined in its Go type. It uses the API version extracted from ctx and the
// provided scheme for validation.
//
// The ctx MUST contain requestInfo, which determines the target API for
// validation. The obj is converted to the API version using the provided scheme
// before validation occurs. The scheme MUST have the declarative validation
// registered for the requested resource/subresource.
//
// option should contain any validation options that the declarative validation
// tags expect.
//
// Returns a field.ErrorList containing any validation errors. An internal error
// is included if requestInfo is missing from the context or if version
// conversion fails.
func ValidateDeclaratively(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj runtime.Object) field.ErrorList {
	if requestInfo, found := genericapirequest.RequestInfoFrom(ctx); found {
		groupVersion := schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}
		versionedObj, err := scheme.ConvertToVersion(obj, groupVersion)
		if err != nil {
			return field.ErrorList{field.InternalError(nil, fmt.Errorf("unexpected error converting to versioned type: %w", err))}
		}
		subresources, err := parseSubresourcePath(requestInfo.Subresource)
		if err != nil {
			return field.ErrorList{field.InternalError(nil, fmt.Errorf("unexpected error parsing subresource path: %w", err))}
		}
		return scheme.Validate(ctx, options, versionedObj, subresources...)
	} else {
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("could not find requestInfo in context"))}
	}
}

// ValidateUpdateDeclaratively validates obj and oldObj against declarative
// validation tags defined in its Go type. It uses the API version extracted from
// ctx and the provided scheme for validation.
//
// The ctx MUST contain requestInfo, which determines the target API for
// validation. The obj is converted to the API version using the provided scheme
// before validation occurs. The scheme MUST have the declarative validation
// registered for the requested resource/subresource.
//
// option should contain any validation options that the declarative validation
// tags expect.
//
// Returns a field.ErrorList containing any validation errors. An internal error
// is included if requestInfo is missing from the context or if version
// conversion fails.
func ValidateUpdateDeclaratively(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj, oldObj runtime.Object) field.ErrorList {
	if requestInfo, found := genericapirequest.RequestInfoFrom(ctx); found {
		groupVersion := schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}
		versionedObj, err := scheme.ConvertToVersion(obj, groupVersion)
		if err != nil {
			return field.ErrorList{field.InternalError(nil, fmt.Errorf("unexpected error converting to versioned type: %w", err))}
		}
		versionedOldObj, err := scheme.ConvertToVersion(oldObj, groupVersion)
		if err != nil {
			return field.ErrorList{field.InternalError(nil, fmt.Errorf("unexpected error converting to versioned type: %w", err))}
		}
		subresources, err := parseSubresourcePath(requestInfo.Subresource)
		if err != nil {
			return field.ErrorList{field.InternalError(nil, fmt.Errorf("unexpected error parsing subresource path: %w", err))}
		}
		return scheme.ValidateUpdate(ctx, options, versionedObj, versionedOldObj, subresources...)
	} else {
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("could not find requestInfo in context"))}
	}
}

func parseSubresourcePath(subresourcePath string) ([]string, error) {
	if len(subresourcePath) == 0 {
		return nil, nil
	}
	if subresourcePath[0] != '/' {
		return nil, fmt.Errorf("invalid subresource path: %s", subresourcePath)
	}
	parts := strings.Split(subresourcePath[1:], "/")
	return parts, nil
}

// CompareDeclarativeErrorsAndEmitMismatches checks for mismatches between imperative and declarative validation
// and logs + emits metrics when inconsistencies are found
func CompareDeclarativeErrorsAndEmitMismatches(imperativeErrs, declarativeErrs field.ErrorList) {
	mismatchDetails := gatherDeclarativeValidationMismatches(imperativeErrs, declarativeErrs)
	for _, detail := range mismatchDetails {
		// Log information about the mismatch
		klog.Warning(detail)

		// Increment the metric for the mismatch
		validationmetrics.Metrics.EmitDeclarativeValidationMismatchMetric()
	}
}

// gatherDeclarativeValidationMismatches compares imperative and declarative validation errors
// and returns detailed information about any mismatches found. Errors are compared via type, field, and origin
func gatherDeclarativeValidationMismatches(imperativeErrs, declarativeErrs field.ErrorList) []string {
	var mismatchDetails []string
	// short circuit here to minimize allocs for usual case of 0 validation errors
	if len(imperativeErrs) == 0 && len(declarativeErrs) == 0 {
		return mismatchDetails
	}
	// recommendation based on takeover status
	recommendation := "This difference should not affect system operation since hand written validation is authoritative."
	if utilfeature.DefaultFeatureGate.Enabled(features.DeclarativeValidationTakeover) {
		recommendation = "Consider disabling the DeclarativeValidationTakeover feature gate."
	}

	matcher := fldtest.ErrorMatcher{}.ByType().ByField().ByOrigin().RequireOriginWhenInvalid()

	imperativeErrMap := make(map[string][]int)
	declarativeErrMap := make(map[string][]int)

	// Hash imperative errors
	for i, iErr := range imperativeErrs {
		key := matcher.Render(iErr)
		imperativeErrMap[key] = append(imperativeErrMap[key], i)
	}

	// Hash declarative errors
	for j, dErr := range declarativeErrs {
		key := matcher.Render(dErr)
		declarativeErrMap[key] = append(declarativeErrMap[key], j)
	}

	// Track which declarative errors have been matched
	matchedDeclarative := make(map[int]bool)

	// For each imperative error that needs coverage, check if it has matching declarative errors
	for key, iIndices := range imperativeErrMap {
		// Only process the first occurrence of each unique error (dedupe)
		iErr := imperativeErrs[iIndices[0]]

		if !iErr.CoveredByDeclarative {
			continue
		}

		// Get any matching declarative errors
		dIndices, found := declarativeErrMap[key]

		if !found || len(dIndices) == 0 {
			mismatchDetails = append(mismatchDetails,
				fmt.Sprintf(
					"Unexpected difference between hand written validation and declarative validation error results, unmatched error(s) found %s. "+
						"This may indicate an issue with declarative validation. %s",
					key,
					recommendation,
				),
			)
		} else {
			// Mark all matching declarative errors as matched (1:many relationship)
			for _, dIdx := range dIndices {
				matchedDeclarative[dIdx] = true
			}
		}
	}

	// Check for any unmatched declarative errors
	for j := range declarativeErrs {
		if !matchedDeclarative[j] {
			mismatchDetails = append(mismatchDetails,
				fmt.Sprintf(
					"Unexpected difference between hand written validation and declarative validation error results, extra error(s) found %s. "+
						"This may indicate an issue with declarative validation. %s",
					matcher.Render(declarativeErrs[j]),
					recommendation,
				),
			)
		}
	}

	return mismatchDetails
}

// createDeclarativeValidationPanicHandler returns a function with panic recovery logic
// that will increment metrics and either log or append errors based on feature gate settings.
func createDeclarativeValidationPanicHandler(errs *field.ErrorList) func() {
	return func() {
		if r := recover(); r != nil {
			// Increment the panic metric counter
			validationmetrics.Metrics.EmitDeclarativeValidationPanicMetric()

			const errorFmt = "panic during declarative validation: %v"
			if utilfeature.DefaultFeatureGate.Enabled(features.DeclarativeValidationTakeover) {
				// If takeover is enabled, output as a validation error as authorative validator panicked and validation should error
				*errs = append(*errs, field.InternalError(nil, fmt.Errorf(errorFmt, r)))
			} else {
				// if takeover not enabled, log the panic as a warning
				klog.Warningf(errorFmt, r)
			}
		}
	}
}

// WithRecover wraps a validation function with panic recovery logic.
// It takes a validation function with the ValidateDeclaratively signature
// and returns a function with the same signature.
// The returned function will execute the wrapped function and handle any panics by
// incrementing the panic metric, and logging an error message
// if DeclarativeValidationTakeover=disabled, and adding a validation error if enabled.
func WithRecover(
	validateFunc func(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj runtime.Object) field.ErrorList,
) func(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj runtime.Object) field.ErrorList {
	return func(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj runtime.Object) (errs field.ErrorList) {
		defer createDeclarativeValidationPanicHandler(&errs)()

		return validateFunc(ctx, options, scheme, obj)
	}
}

// WithRecoverUpdate wraps an update validation function with panic recovery logic.
// It takes a validation function with the ValidateUpdateDeclaratively signature
// and returns a function with the same signature.
// The returned function will execute the wrapped function and handle any panics by
// incrementing the panic metric, and logging an error message
// if DeclarativeValidationTakeover=disabled, and adding a validation error if enabled.
func WithRecoverUpdate(
	validateUpdateFunc func(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj, oldObj runtime.Object) field.ErrorList,
) func(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj, oldObj runtime.Object) field.ErrorList {
	return func(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj, oldObj runtime.Object) (errs field.ErrorList) {
		defer createDeclarativeValidationPanicHandler(&errs)()

		return validateUpdateFunc(ctx, options, scheme, obj, oldObj)
	}
}

// ValidateDeclarativelyWithRecover validates obj against declarative validation tags
// with panic recovery logic. It uses the API version extracted from ctx and the
// provided scheme for validation.
//
// The ctx MUST contain requestInfo, which determines the target API for
// validation. The obj is converted to the API version using the provided scheme
// before validation occurs. The scheme MUST have the declarative validation
// registered for the requested resource/subresource.
//
// option should contain any validation options that the declarative validation
// tags expect.
//
// Returns a field.ErrorList containing any validation errors. An internal error
// is included if requestInfo is missing from the context, if version
// conversion fails, or if a panic occurs during validation when
// DeclarativeValidationTakeover is enabled.
func ValidateDeclarativelyWithRecovery(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj runtime.Object) field.ErrorList {
	return WithRecover(ValidateDeclaratively)(ctx, options, scheme, obj)
}

// ValidateUpdateDeclarativelyWithRecover validates obj and oldObj against declarative
// validation tags with panic recovery logic. It uses the API version extracted from
// ctx and the provided scheme for validation.
//
// The ctx MUST contain requestInfo, which determines the target API for
// validation. The obj is converted to the API version using the provided scheme
// before validation occurs. The scheme MUST have the declarative validation
// registered for the requested resource/subresource.
//
// option should contain any validation options that the declarative validation
// tags expect.
//
// Returns a field.ErrorList containing any validation errors. An internal error
// is included if requestInfo is missing from the context, if version
// conversion fails, or if a panic occurs during validation when
// DeclarativeValidationTakeover is enabled.
func ValidateUpdateDeclarativelyWithRecovery(ctx context.Context, options sets.Set[string], scheme *runtime.Scheme, obj, oldObj runtime.Object) field.ErrorList {
	return WithRecoverUpdate(ValidateUpdateDeclaratively)(ctx, options, scheme, obj, oldObj)
}
