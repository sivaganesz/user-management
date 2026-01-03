package repositories

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Common repository errors
var (
	// ErrNotFound is returned when a document is not found
	ErrNotFound = mongo.ErrNoDocuments

	// ErrDuplicateKey is returned when trying to insert a duplicate document
	ErrDuplicateKey = errors.New("duplicate key error")

	// ErrInvalidInput is returned when the input is invalid
	ErrInvalidInput = errors.New("invalid input")

	// ErrUpdateFailed is returned when an update operation fails
	ErrUpdateFailed = errors.New("update failed")

	// ErrDeleteFailed is returned when a delete operation fails
	ErrDeleteFailed = errors.New("delete failed")
)

// Domain-specific "not found" errors
// These errors wrap mongo.ErrNoDocuments to provide domain context
// Usage in repositories:
//
//	if err == mongo.ErrNoDocuments {
//	    return nil, WrapNotFound(err, ErrDealNotFound)
//	}
var (
	// ErrDealNotFound is returned when a deal is not found
	ErrDealNotFound = errors.New("deal not found")

	// ErrLeadNotFound is returned when a lead is not found
	ErrLeadNotFound = errors.New("lead not found")

	// ErrCustomerNotFound is returned when a customer is not found
	ErrCustomerNotFound = errors.New("customer not found")

	// ErrActivityNotFound is returned when an activity is not found
	ErrActivityNotFound = errors.New("activity not found")

	// ErrCommunicationNotFound is returned when a communication is not found
	ErrCommunicationNotFound = errors.New("communication not found")

	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrTemplateNotFound is returned when a template is not found
	ErrTemplateNotFound = errors.New("template not found")

	// ErrCampaignNotFound is returned when a campaign is not found
	ErrCampaignNotFound = errors.New("campaign not found")

	// ErrQuestionnaireNotFound is returned when a questionnaire is not found
	ErrQuestionnaireNotFound = errors.New("questionnaire not found")

	// ErrDocumentNotFound is returned when a document is not found
	ErrDocumentNotFound = errors.New("document not found")
)

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsDuplicateKey checks if an error is a duplicate key error
func IsDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	return mongo.IsDuplicateKeyError(err) || errors.Is(err, ErrDuplicateKey)
}


// IsCustomerNotFound checks if an error indicates a customer was not found
func IsCustomerNotFound(err error) bool {
	return errors.Is(err, ErrCustomerNotFound)
}

// IsActivityNotFound checks if an error indicates an activity was not found
func IsActivityNotFound(err error) bool {
	return errors.Is(err, ErrActivityNotFound)
}

// IsCommunicationNotFound checks if an error indicates a communication was not found
func IsCommunicationNotFound(err error) bool {
	return errors.Is(err, ErrCommunicationNotFound)
}

// IsUserNotFound checks if an error indicates a user was not found
func IsUserNotFound(err error) bool {
	return errors.Is(err, ErrUserNotFound)
}

// IsTemplateNotFound checks if an error indicates a template was not found
func IsTemplateNotFound(err error) bool {
	return errors.Is(err, ErrTemplateNotFound)
}


// IsDocumentNotFound checks if an error indicates a document was not found
func IsDocumentNotFound(err error) bool {
	return errors.Is(err, ErrDocumentNotFound)
}

// WrapNotFound wraps mongo.ErrNoDocuments with a domain-specific error
// This preserves the original MongoDB error while adding domain context
//
// Usage in repository methods:
//
//	var deal models.Deal
//	err := r.collection.FindOne(ctx, filter).Decode(&deal)
//	if err == mongo.ErrNoDocuments {
//	    return nil, WrapNotFound(err, ErrDealNotFound)
//	}
//	if err != nil {
//	    return nil, fmt.Errorf("failed to get deal: %w", err)
//	}
//	return &deal, nil
//
// This allows handlers to check:
//
//	if IsDealNotFound(err) { ... }  // domain-specific check
//	if IsNotFound(err) { ... }      // generic not found check
func WrapNotFound(err error, domainErr error) error {
	if err == nil {
		return nil
	}
	// Only wrap if it's actually a "not found" error
	if errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("%w: %w", domainErr, err)
	}
	// Return original error if it's not a "not found" error
	return err
}
