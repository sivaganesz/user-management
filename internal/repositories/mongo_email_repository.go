package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoEmailRepository handles email-specific communication operations with MongoDB
type MongoEmailRepository struct {
	client                *mongodb.Client
	messagesCollection    *mongo.Collection
	threadsCollection     *mongo.Collection
	attachmentsCollection *mongo.Collection
}

func NewMongoEmailRepository(client *mongodb.Client) *MongoEmailRepository {
	return &MongoEmailRepository{
		client: client,
		messagesCollection: client.Collection("communication"),
		threadsCollection:  client.Collection("message_threads"),
		attachmentsCollection: client.Collection("message_attachments"),
	}
}

// EmailFilters represents filters for email inbox queries
type EmailFilters struct {
	Channel    string
	Status     string
	IsRead     *bool
	IsStarred  *bool
	DateFrom   *time.Time
	DateTo     *time.Time
	EntityType string
	EntityID   primitive.ObjectID
	Limit      int
}


func (r *MongoEmailRepository) CreateMessage(ctx context.Context, message *models.MongoCommunication) error {
	message.Channel = string(models.CommunicationChannelEmail)
	result, err := r.messagesCollection.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("error creating email message: %w", err)
	}
	message.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// CreateMessageFromCommMessage creates a message from CommMessage type (backward compatibility)
// Converts CommMessage to MongoCommunication and inserts it
func (r *MongoEmailRepository) CreateMessageFromCommMessage(msg *models.CommMessage) error {
	// Convert CommMessage to MongoCommunication
	mongoMsg := &models.MongoCommunication{
		ID:         msg.MessageID,
		ThreadID:   msg.ThreadID,
		Channel:    msg.Channel,
		Direction:  msg.Direction,
		From:       msg.FromAddress,
		To:         "", // Will be set from first ToAddress
		Subject:    msg.Subject,
		Body:       msg.BodyText,
		CustomerID: primitive.NilObjectID, // Set from EntityID if EntityType is customer
		DealID:     primitive.NilObjectID, // Set from EntityID if EntityType is deal
		Status:     msg.Status,
		SentAt:     msg.CreatedAt,
		ReadAt:     nil,
		CreatedAt:  msg.CreatedAt,
		UpdatedAt:  msg.UpdatedAt,
	}

	// Set To address
	if len(msg.ToAddresses) > 0 {
		mongoMsg.To = msg.ToAddresses[0]
	}

	// Set entity IDs based on EntityType
	if msg.EntityType == "customer" {
		mongoMsg.CustomerID = msg.EntityID
	}
	return r.CreateMessage(context.Background(), mongoMsg)
}

// GetMessageByID retrieves an email message by ID
func (r *MongoEmailRepository) GetMessageByID(ctx context.Context, id primitive.ObjectID) (*models.MongoCommunication, error) {
	var msg models.MongoCommunication

	filter := bson.M{
		"_id":     id,
		"channel": string(models.CommunicationChannelEmail),
	}

	err := r.messagesCollection.FindOne(ctx, filter).Decode(&msg)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, WrapNotFound(mongo.ErrNoDocuments, ErrCommunicationNotFound)
		}
		return nil, fmt.Errorf("error finding email message: %w", err)
	}

	return &msg, nil
}

// GetMessagesByThread retrieves all email messages in a thread
func (r *MongoEmailRepository) GetMessagesByThread(ctx context.Context, threadID primitive.ObjectID) ([]*models.MongoCommunication, error) {
	filter := bson.M{
		"thread_id": threadID,
		"channel":   string(models.CommunicationChannelEmail),
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "sent_at", Value: 1}}) // Chronological order

	cursor, err := r.messagesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error retrieving email thread: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.MongoCommunication
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("error decoding email messages: %w", err)
	}
	return messages, nil
}

// GetInbox retrieves inbox messages for a user with filters
func (r *MongoEmailRepository) GetInbox(ctx context.Context, userID primitive.ObjectID, filters EmailFilters) ([]*models.MongoCommunication, error) {
	if filters.Limit <= 0 {
		filters.Limit = 50 // Default limit
	}

	filter := bson.M{
		"channel": string(models.CommunicationChannelEmail),
	}
	// Apply filters
	if filters.Status != "" {
		filter["status"] = filters.Status
	}
	if filters.IsRead != nil {
		if *filters.IsRead {
			filter["read_at"] = bson.M{"exists":true, "$ne":nil}
		} else {
			filter["read_at"] = bson.M{"$exists": false}
		}
	}
	if filters.IsStarred != nil {
		filter["isStarred"] = *filters.IsStarred
	}
	if filters.DateFrom != nil {
		filter["sentAt"] = bson.M{"$gte": filters.DateFrom}
	}

	if filters.DateTo != nil {
		if sentAtFilter, ok := filter["sentAt"].(bson.M); ok {
			sentAtFilter["$lte"] = filters.DateTo
		}else {
			filter["sentAt"] = bson.M{"$lte": filters.DateTo}
		}
	}
	if !filters.EntityID.IsZero() {
		if filters.EntityType == "customer" {
			filter["customerId"] = filters.EntityID
		} 
	}

	opts := options.Find().
		SetLimit(int64(filters.Limit)).
		SetSort(bson.D{{Key: "sent_at", Value: -1}})
	cursor, err := r.messagesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error retrieving email inbox: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.MongoCommunication
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("error decoding email messages: %w", err)
	}

	return messages, nil


}

// UpdateMessageStatus updates email message status
func (r *MongoEmailRepository) UpdateMessageStatus(ctx context.Context, id primitive.ObjectID, status string) error {
	filter := bson.M{
		"_id":     id,
		"channel": string(models.CommunicationChannelEmail),
	}
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	result, err := r.messagesCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating email status: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrCommunicationNotFound)
	}

	return nil
}

//MarkAsRead marks an email message as read
func (r *MongoEmailRepository) MarkAsRead(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{
		"_id": id,
		"channel": string(models.CommunicationChannelEmail),
	}
	update := bson.M{
		"$set" : bson.M{
			"read_at": time.Now(),
			"status": string(models.CommunicationStatusRead),
		},
	}
	result, err := r.messagesCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error marking email as read: %w", err)
	}
	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrCommunicationNotFound)
	}
	return nil
}

// MessageThread represents an email conversation thread
type MessageThread struct {
	ID                   primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Subject              string               `bson:"subject" json:"subject"`
	Channel              string               `bson:"channel" json:"channel"`
	EntityType           string               `bson:"entityType,omitempty" json:"entityType,omitempty"`
	EntityID             primitive.ObjectID   `bson:"entityId,omitempty" json:"entityId,omitempty"`
	ParticipantAddresses []string             `bson:"participantAddresses" json:"participantAddresses"`
	ParticipantIDs       []primitive.ObjectID `bson:"participantIds,omitempty" json:"participantIds,omitempty"`
	MessageCount         int                  `bson:"messageCount" json:"messageCount"`
	UnreadCount          int                  `bson:"unreadCount" json:"unreadCount"`
	LastMessageAt        time.Time            `bson:"lastMessageAt" json:"lastMessageAt"`
	LastMessageSnippet   string               `bson:"lastMessageSnippet" json:"lastMessageSnippet"`
	IsArchived           bool                 `bson:"isArchived" json:"isArchived"`
	CreatedAt            time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time            `bson:"updatedAt" json:"updatedAt"`
}

// CreateThread creates a new message thread
func (r *MongoEmailRepository) CreateThread(ctx context.Context, thread *MessageThread) error {
	thread.Channel = string(models.CommunicationChannelEmail)
	thread.CreatedAt = time.Now()
	thread.UpdatedAt = time.Now()
	result, err := r.threadsCollection.InsertOne(ctx, thread)
	if err != nil {
		return fmt.Errorf("error creating email thread: %w", err)
	}
	thread.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetThreadByID retrieves a thread by ID
func (r *MongoEmailRepository) GetThreadByID(ctx context.Context, threadID primitive.ObjectID) (*MessageThread, error) {
	filter := bson.M{
		"_id":     threadID,
		"channel": string(models.CommunicationChannelEmail),
	}
	var thread MessageThread
	err := r.threadsCollection.FindOne(ctx, filter).Decode(&thread)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, WrapNotFound(mongo.ErrNoDocuments, ErrCommunicationNotFound)
		}
		return nil, fmt.Errorf("error finding thread: %w", err)
	}

	return &thread, nil
}

// UpdateThreadMetadata updates thread metadata with latest message info
func (r *MongoEmailRepository) UpdateThreadMetadata(ctx context.Context, threadID primitive.ObjectID, lastMessage *models.MongoCommunication) error {
	// Get current thread
	thread, err := r.GetThreadByID(ctx, threadID)
	if err != nil {
		return err
	}

	// Update counts
	thread.MessageCount++
	if lastMessage.ReadAt == nil {
		thread.UnreadCount++
	}
	thread.LastMessageAt = lastMessage.SentAt
	thread.LastMessageSnippet = generateSnippet(lastMessage.Body)
	thread.UpdatedAt = time.Now()

	// Save updated thread
	filter := bson.M{"_id": threadID}
	update := bson.M{
		"$set": bson.M{
			"messageCount":       thread.MessageCount,
			"unreadCount":        thread.UnreadCount,
			"lastMessageAt":      thread.LastMessageAt,
			"lastMessageSnippet": thread.LastMessageSnippet,
			"updated_at":         thread.UpdatedAt,
		},
	}

	result, err := r.threadsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating thread metadata: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrCommunicationNotFound)
	}

	return nil
}

// generateSnippet creates a short preview of the message body
func generateSnippet(body string) string {
	maxLen := 100
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "..."
}

// MessageAttachment represents an email attachment
type MessageAttachment struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MessageID       primitive.ObjectID `bson:"messageId" json:"messageId"`
	FileName        string             `bson:"fileName" json:"fileName"`
	FileSizeBytes   int64              `bson:"fileSizeBytes" json:"fileSizeBytes"`
	MimeType        string             `bson:"mimeType" json:"mimeType"`
	ContentID       string             `bson:"contentId,omitempty" json:"contentId,omitempty"`
	StorageLocation string             `bson:"storageLocation" json:"storageLocation"`
	IsInline        bool               `bson:"isInline" json:"isInline"`
	DownloadCount   int                `bson:"downloadCount" json:"downloadCount"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
}

// SaveAttachment saves message attachment metadata
func (r *MongoEmailRepository) SaveAttachment(ctx context.Context, att *MessageAttachment) error {
	att.CreatedAt = time.Now()
	att.DownloadCount = 0

	result, err := r.attachmentsCollection.InsertOne(ctx, att)
	if err != nil {
		return fmt.Errorf("error saving attachment: %w", err)
	}

	att.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetAttachmentsByMessage retrieves all attachments for a message
func (r *MongoEmailRepository) GetAttachmentsByMessage(ctx context.Context, messageID primitive.ObjectID) ([]*MessageAttachment, error) {
	filter := bson.M{"message_id": messageID}

	cursor, err := r.attachmentsCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error retrieving attachments: %w", err)
	}
	defer cursor.Close(ctx)

	var attachments []*MessageAttachment
	if err := cursor.All(ctx, &attachments); err != nil {
		return nil, fmt.Errorf("error decoding attachments: %w", err)
	}

	return attachments, nil
}

// IncrementAttachmentDownloadCount increments download count for an attachment
func (r *MongoEmailRepository) IncrementAttachmentDownloadCount(ctx context.Context, attachmentID primitive.ObjectID) error {
	filter := bson.M{"_id": attachmentID}
	update := bson.M{
		"$inc": bson.M{"downloadCount": 1},
	}

	result, err := r.attachmentsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error incrementing download count: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrCommunicationNotFound)
	}

	return nil
}

// EnsureIndexes creates the required indexes for email collections
func (r *MongoEmailRepository) EnsureIndexes(ctx context.Context) error {
	// Message threads indexes
	threadIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "channel", Value: 1},
				{Key: "lastMessageAt", Value: -1},
			},
		},
		{
			Keys: bson.D{{Key: "entity_id", Value: 1}},
		},
	}

	_, err := r.threadsCollection.Indexes().CreateMany(ctx, threadIndexes)
	if err != nil {
		return fmt.Errorf("error creating thread indexes: %w", err)
	}

	// Message attachments indexes
	attachmentIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "message_id", Value: 1}},
		},
	}

	_, err = r.attachmentsCollection.Indexes().CreateMany(ctx, attachmentIndexes)
	if err != nil {
		return fmt.Errorf("error creating attachment indexes: %w", err)
	}

	return nil
}

// =============================================================================
// Service layer compatibility methods (without context parameter)
// =============================================================================

// GetThreadByIDCompat retrieves a thread by ID without context, returning *models.MessageThread
func (r *MongoEmailRepository) GetThreadByIDCompat(threadID primitive.ObjectID) (*models.MessageThread, error) {
	ctx := context.Background()
	repoThread, err := r.GetThreadByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	// Convert repository thread to model thread
	return &models.MessageThread{
		ThreadID:             repoThread.ID,
		Subject:              repoThread.Subject,
		Channel:              repoThread.Channel,
		EntityType:           repoThread.EntityType,
		EntityID:             repoThread.EntityID,
		ParticipantAddresses: repoThread.ParticipantAddresses,
		MessageCount:         repoThread.MessageCount,
		UnreadCount:          repoThread.UnreadCount,
		LastMessageAt:        repoThread.LastMessageAt,
		LastMessageSnippet:   repoThread.LastMessageSnippet,
		IsArchived:           repoThread.IsArchived,
		CreatedAt:            repoThread.CreatedAt,
		UpdatedAt:            repoThread.UpdatedAt,
	}, nil
}


// CreateThreadCompat creates a thread without context, accepting *models.MessageThread
func (r *MongoEmailRepository) CreateThreadCompat(thread *models.MessageThread) error {
	ctx := context.Background()
	repoThread := &MessageThread{
		ID:                   thread.ThreadID,
		Subject:              thread.Subject,
		Channel:              thread.Channel,
		EntityType:           thread.EntityType,
		EntityID:             thread.EntityID,
		ParticipantAddresses: thread.ParticipantAddresses,
		MessageCount:         thread.MessageCount,
		UnreadCount:          thread.UnreadCount,
		LastMessageAt:        thread.LastMessageAt,
		LastMessageSnippet:   thread.LastMessageSnippet,
		IsArchived:           thread.IsArchived,
		CreatedAt:            thread.CreatedAt,
		UpdatedAt:            thread.UpdatedAt,
	}
	return r.CreateThread(ctx, repoThread)
}

// UpdateThreadMetadataCompat updates thread metadata without context, accepting *models.CommMessage
func (r *MongoEmailRepository) UpdateThreadMetadataCompat(threadID primitive.ObjectID, lastMessage *models.CommMessage) error {
	ctx := context.Background()
	// Convert CommMessage to MongoCommunication for the underlying method
	mongoComm := &models.MongoCommunication{
		ID:        lastMessage.MessageID,
		Channel:   lastMessage.Channel,
		Subject:   lastMessage.Subject,
		Body:      lastMessage.Snippet, // Use snippet as body
		CreatedAt: lastMessage.CreatedAt,
	}
	return r.UpdateThreadMetadata(ctx, threadID, mongoComm)
}

// GetMessagesByThreadCompat retrieves messages by thread without context, returning []*models.CommMessage
func (r *MongoEmailRepository) GetMessagesByThreadCompat(threadID primitive.ObjectID) ([]*models.CommMessage, error) {
	ctx := context.Background()
	mongoMessages, err := r.GetMessagesByThread(ctx, threadID)
	if err != nil {
		return nil, err
	}
	// Convert MongoCommunication to CommMessage
	result := make([]*models.CommMessage, len(mongoMessages))
	for i, mc := range mongoMessages {
		result[i] = &models.CommMessage{
			MessageID:   mc.ID,
			Channel:     mc.Channel,
			Subject:     mc.Subject,
			Snippet:     mc.Body, // Use body as snippet
			FromAddress: mc.From,
			ToAddresses: []string{mc.To}, // Single To becomes array
			CreatedAt:   mc.CreatedAt,
		}
	}
	return result, nil
}

// GetMessageByIDCompat retrieves a message by ID and converts to CommMessage (service layer compatibility)
func (r *MongoEmailRepository) GetMessageByIDCompat(messageID primitive.ObjectID) (*models.CommMessage, error) {
	ctx := context.Background()
	mongoMsg, err := r.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Convert MongoCommunication to CommMessage
	return &models.CommMessage{
		MessageID:   mongoMsg.ID,
		EntityType:  "customer",
		EntityID:    mongoMsg.CustomerID,
		Channel:     mongoMsg.Channel,
		Direction:   mongoMsg.Direction,
		Subject:     mongoMsg.Subject,
		BodyText:    mongoMsg.Body,
		Status:      mongoMsg.Status,
		FromAddress: mongoMsg.From,
		ToAddresses: []string{mongoMsg.To},
		CreatedAt:   mongoMsg.CreatedAt,
	}, nil
}

// UpdateMessageStatusCompat updates message status (service layer compatibility - no context)
func (r *MongoEmailRepository) UpdateMessageStatusCompat(messageID primitive.ObjectID, status string) error {
	return r.UpdateMessageStatus(context.Background(), messageID, status)
}

// GetInboxCompat retrieves inbox messages with filters (service layer compatibility - no context)
func (r *MongoEmailRepository) GetInboxCompat(userID primitive.ObjectID, filters EmailFilters) ([]*models.CommMessage, error) {
	mongoMsgs, err := r.GetInbox(context.Background(), userID, filters)
	if err != nil {
		return nil, err
	}

	// Convert MongoCommunication to CommMessage
	result := make([]*models.CommMessage, len(mongoMsgs))
	for i, m := range mongoMsgs {
		result[i] = &models.CommMessage{
			MessageID:   m.ID,
			EntityType:  "customer",
			EntityID:    m.CustomerID,
			Channel:     m.Channel,
			Direction:   m.Direction,
			Subject:     m.Subject,
			BodyText:    m.Body,
			Status:      m.Status,
			FromAddress: m.From,
			ToAddresses: []string{m.To},
			IsRead:      m.IsRead, // Use field directly
			CreatedAt:   m.CreatedAt,
		}
	}
	return result, nil
}

// MarkAsReadCompat marks a message as read (service layer compatibility - no context)
func (r *MongoEmailRepository) MarkAsReadCompat(messageID primitive.ObjectID) error {
	return r.MarkAsRead(context.Background(), messageID)
}

// CreateMessageCompat creates a message from CommMessage (service layer compatibility - no context)
func (r *MongoEmailRepository) CreateMessageCompat(msg *models.CommMessage) error {
	mongoMsg := &models.MongoCommunication{
		ID:        msg.MessageID,
		ThreadID:  msg.ThreadID,
		Channel:   msg.Channel,
		Direction: msg.Direction,
		Subject:   msg.Subject,
		Body:      msg.BodyText,
		BodyHTML:  msg.BodyHTML,
		Snippet:   msg.Snippet,
		Priority:  msg.Priority,
		Status:    msg.Status,
		From:      msg.FromName,
		FromEmail: msg.FromAddress,
		CC:        msg.CCAddresses,
		BCC:       msg.BCCAddresses,
		UserID:    msg.UserID,
		IsRead:    msg.IsRead,
		IsStarred: msg.IsStarred,
		IsArchived: msg.IsArchived,
		CreatedAt: msg.CreatedAt,
		UpdatedAt: time.Now(),
	}

	// Set primary recipient
	if len(msg.ToAddresses) > 0 {
		mongoMsg.To = msg.ToAddresses[0]
		mongoMsg.ToEmail = msg.ToAddresses[0]
	}

	// Map EntityID to correct field based on EntityType
	if msg.EntityType == "customer" {
		mongoMsg.CustomerID = msg.EntityID
		// Look up customer to get company name
		customer, err := r.lookupCustomer(msg.EntityID)
		if err == nil && customer != nil {
			mongoMsg.Company = customer.Company
		}
	}

	// Set SentAt for outbound messages
	if msg.Direction == models.DirectionOutbound {
		mongoMsg.SentAt = msg.CreatedAt
		if msg.SentAt != nil {
			mongoMsg.SentAt = *msg.SentAt
		}
	}

	// Set ReadAt if message is marked as read
	if msg.IsRead {
		now := time.Now()
		mongoMsg.ReadAt = &now
	}

	// Copy attachments if present
	if len(msg.Attachments) > 0 {
		mongoMsg.Attachments = msg.Attachments
	}

	return r.CreateMessage(context.Background(), mongoMsg)
}


// CustomerLookupResult holds customer lookup results
type CustomerLookupResult struct {
	Company string `bson:"company"`
}

// lookupCustomer retrieves customer info for populating communication fields
func (r *MongoEmailRepository) lookupCustomer(customerID primitive.ObjectID) (*CustomerLookupResult, error) {
	if customerID.IsZero() {
		return nil, nil
	}
	collection := r.client.Database().Collection("customers")
	var customer CustomerLookupResult
	err := collection.FindOne(context.Background(), bson.M{"_id": customerID}).Decode(&customer)
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// SaveAttachmentCompat saves an attachment from models.MessageAttachment (service layer compatibility)
func (r *MongoEmailRepository) SaveAttachmentCompat(att *models.MessageAttachment) error {
	repoAtt := &MessageAttachment{
		ID:              att.AttachmentID,
		MessageID:       att.MessageID,
		FileName:        att.FileName,
		FileSizeBytes:   att.FileSizeBytes,
		MimeType:        att.MimeType,
		ContentID:       att.ContentID,
		StorageLocation: att.StorageLocation,
		IsInline:        att.IsInline,
		DownloadCount:   att.DownloadCount,
		CreatedAt:       att.CreatedAt,
	}
	return r.SaveAttachment(context.Background(), repoAtt)
}

// IncrementAttachmentDownloadCountCompat increments attachment download count (service layer compatibility)
func (r *MongoEmailRepository) IncrementAttachmentDownloadCountCompat(attachmentID primitive.ObjectID) error {
	return r.IncrementAttachmentDownloadCount(context.Background(), attachmentID)
}


// DeleteMessage deletes a message by ID (used for rollback on scheduling failure)
func (r *MongoEmailRepository) DeleteMessage(ctx context.Context, messageID primitive.ObjectID) error {
	filter := bson.M{"_id": messageID}
	result, err := r.messagesCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting message: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("message not found: %s", messageID.Hex())
	}
	return nil
}

// DeleteMessageCompat deletes a message by ID (service layer compatibility)
func (r *MongoEmailRepository) DeleteMessageCompat(messageID primitive.ObjectID) error {
	return r.DeleteMessage(context.Background(), messageID)
}
