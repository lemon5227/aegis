package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	outboxMessageTypePost             = "POST"
	outboxMessageTypeComment          = "COMMENT"
	outboxMessageTypeProfileUpdate    = "PROFILE_UPDATE"
	outboxMessageTypeFavorite         = "FAVORITE_OP"
	outboxMessageTypeGovernance       = "GOVERNANCE"
	outboxMessageTypePostVoteSet      = "POST_VOTE_SET"
	outboxMessageTypeCommentVoteSet   = "COMMENT_VOTE_SET"
	outboxMessageTypeGovernancePolicy = "GOVERNANCE_POLICY_UPDATE"
)

func normalizeIncomingMessageForSignature(message IncomingMessage) IncomingMessage {
	message.Type = strings.ToUpper(strings.TrimSpace(message.Type))
	message.OpType = normalizeOperationType(message.OpType, postOpTypeCreate)
	message.OpID = strings.TrimSpace(message.OpID)
	message.AuthScope = normalizeAuthScope(message.AuthScope)
	message.ID = strings.TrimSpace(message.ID)
	message.Pubkey = strings.TrimSpace(message.Pubkey)
	message.VoterPubkey = strings.TrimSpace(message.VoterPubkey)
	message.VoteState = strings.ToUpper(strings.TrimSpace(message.VoteState))
	message.PostID = strings.TrimSpace(message.PostID)
	message.CommentID = strings.TrimSpace(message.CommentID)
	message.ParentID = strings.TrimSpace(message.ParentID)
	message.DisplayName = strings.TrimSpace(message.DisplayName)
	message.AvatarURL = strings.TrimSpace(message.AvatarURL)
	message.Title = strings.TrimSpace(message.Title)
	message.Body = strings.TrimSpace(message.Body)
	message.ContentCID = strings.TrimSpace(message.ContentCID)
	message.ImageCID = strings.TrimSpace(message.ImageCID)
	message.ThumbCID = strings.TrimSpace(message.ThumbCID)
	message.ImageMIME = strings.TrimSpace(message.ImageMIME)
	message.SubID = normalizeSubID(message.SubID)
	message.Signature = strings.TrimSpace(message.Signature)
	message.TargetPubkey = strings.TrimSpace(message.TargetPubkey)
	message.AdminPubkey = strings.TrimSpace(message.AdminPubkey)
	message.Reason = strings.TrimSpace(message.Reason)
	return message
}

func signedIncomingMessageType(messageType string) bool {
	switch strings.ToUpper(strings.TrimSpace(messageType)) {
	case "POST", "COMMENT", "POST_DELETE", "COMMENT_DELETE",
		"PROFILE_UPDATE", "POST_UPVOTE", "POST_DOWNVOTE", "POST_VOTE_SET",
		"COMMENT_UPVOTE", "COMMENT_DOWNVOTE", "COMMENT_VOTE_SET",
		"GOVERNANCE_POLICY_UPDATE", "SHADOW_BAN", "UNBAN":
		return true
	default:
		return false
	}
}

func signerPubkeyForIncomingMessage(message IncomingMessage) (string, error) {
	switch message.Type {
	case "POST", "COMMENT", "POST_DELETE", "COMMENT_DELETE", "PROFILE_UPDATE":
		if message.Pubkey == "" {
			return "", errors.New("message pubkey is required")
		}
		return message.Pubkey, nil
	case "POST_UPVOTE", "POST_DOWNVOTE", "POST_VOTE_SET", "COMMENT_UPVOTE", "COMMENT_DOWNVOTE", "COMMENT_VOTE_SET":
		voterPubkey := strings.TrimSpace(message.VoterPubkey)
		if voterPubkey == "" {
			voterPubkey = message.Pubkey
		}
		if voterPubkey == "" {
			return "", errors.New("voter pubkey is required")
		}
		return voterPubkey, nil
	case "GOVERNANCE_POLICY_UPDATE", "SHADOW_BAN", "UNBAN":
		if message.AdminPubkey == "" {
			return "", errors.New("admin pubkey is required")
		}
		return message.AdminPubkey, nil
	default:
		return "", fmt.Errorf("message type %s does not support signatures", message.Type)
	}
}

func buildIncomingMessageSignaturePayload(message IncomingMessage) (string, error) {
	message = normalizeIncomingMessageForSignature(message)
	if !signedIncomingMessageType(message.Type) {
		return "", fmt.Errorf("message type %s does not require signature payload", message.Type)
	}
	if _, err := signerPubkeyForIncomingMessage(message); err != nil {
		return "", err
	}
	if message.Timestamp <= 0 {
		return "", errors.New("message timestamp is required")
	}

	switch message.Type {
	case "POST":
		return fmt.Sprintf(
			"type=%s|op_type=%s|op_id=%s|schema=%d|scope=%s|id=%s|pubkey=%s|display_name=%s|avatar_url=%s|title=%s|body=%s|content_cid=%s|image_cid=%s|thumb_cid=%s|image_mime=%s|image_size=%d|image_width=%d|image_height=%d|sub_id=%s|timestamp=%d|lamport=%d",
			message.Type, message.OpType, message.OpID, message.SchemaVersion, message.AuthScope, message.ID, message.Pubkey,
			message.DisplayName, message.AvatarURL, message.Title, message.Body, message.ContentCID, message.ImageCID,
			message.ThumbCID, message.ImageMIME, message.ImageSize, message.ImageWidth, message.ImageHeight, message.SubID,
			message.Timestamp, message.Lamport,
		), nil
	case "COMMENT":
		attachmentsJSON, err := encodeCommentAttachmentsJSON(normalizeCommentAttachments(message.CommentAttachments))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(
			"type=%s|op_type=%s|op_id=%s|schema=%d|scope=%s|id=%s|pubkey=%s|post_id=%s|parent_id=%s|display_name=%s|avatar_url=%s|body=%s|attachments=%s|timestamp=%d|lamport=%d",
			message.Type, message.OpType, message.OpID, message.SchemaVersion, message.AuthScope, message.ID, message.Pubkey,
			message.PostID, message.ParentID, message.DisplayName, message.AvatarURL, message.Body, attachmentsJSON,
			message.Timestamp, message.Lamport,
		), nil
	case "POST_DELETE":
		return fmt.Sprintf(
			"type=%s|op_type=%s|op_id=%s|schema=%d|scope=%s|pubkey=%s|post_id=%s|timestamp=%d|lamport=%d|deleted_at_lamport=%d",
			message.Type, message.OpType, message.OpID, message.SchemaVersion, message.AuthScope, message.Pubkey,
			message.PostID, message.Timestamp, message.Lamport, message.DeletedAtLamport,
		), nil
	case "COMMENT_DELETE":
		return fmt.Sprintf(
			"type=%s|op_type=%s|op_id=%s|schema=%d|scope=%s|pubkey=%s|post_id=%s|comment_id=%s|timestamp=%d|lamport=%d|deleted_at_lamport=%d",
			message.Type, message.OpType, message.OpID, message.SchemaVersion, message.AuthScope, message.Pubkey,
			message.PostID, message.CommentID, message.Timestamp, message.Lamport, message.DeletedAtLamport,
		), nil
	case "PROFILE_UPDATE":
		return fmt.Sprintf(
			"type=%s|op_id=%s|pubkey=%s|display_name=%s|avatar_url=%s|timestamp=%d",
			message.Type, message.OpID, message.Pubkey, message.DisplayName, message.AvatarURL, message.Timestamp,
		), nil
	case "POST_UPVOTE", "POST_DOWNVOTE", "POST_VOTE_SET":
		voterPubkey, _ := signerPubkeyForIncomingMessage(message)
		return fmt.Sprintf(
			"type=%s|op_id=%s|pubkey=%s|voter_pubkey=%s|post_id=%s|vote_state=%s|timestamp=%d",
			message.Type, message.OpID, message.Pubkey, voterPubkey, message.PostID, message.VoteState, message.Timestamp,
		), nil
	case "COMMENT_UPVOTE", "COMMENT_DOWNVOTE", "COMMENT_VOTE_SET":
		voterPubkey, _ := signerPubkeyForIncomingMessage(message)
		return fmt.Sprintf(
			"type=%s|op_id=%s|pubkey=%s|voter_pubkey=%s|post_id=%s|comment_id=%s|vote_state=%s|timestamp=%d",
			message.Type, message.OpID, message.Pubkey, voterPubkey, message.PostID, message.CommentID, message.VoteState, message.Timestamp,
		), nil
	case "GOVERNANCE_POLICY_UPDATE":
		return fmt.Sprintf(
			"type=%s|op_id=%s|admin_pubkey=%s|hide_history_on_shadowban=%t|timestamp=%d|lamport=%d",
			message.Type, message.OpID, message.AdminPubkey, message.HideHistoryOnShadowBan, message.Timestamp, message.Lamport,
		), nil
	case "SHADOW_BAN", "UNBAN":
		return fmt.Sprintf(
			"type=%s|op_id=%s|admin_pubkey=%s|target_pubkey=%s|reason=%s|timestamp=%d|lamport=%d",
			message.Type, message.OpID, message.AdminPubkey, message.TargetPubkey, message.Reason, message.Timestamp, message.Lamport,
		), nil
	default:
		return "", fmt.Errorf("unsupported signed message type %s", message.Type)
	}
}

func (a *App) signIncomingMessage(message IncomingMessage) (IncomingMessage, error) {
	message = normalizeIncomingMessageForSignature(message)
	signerPubkey, err := signerPubkeyForIncomingMessage(message)
	if err != nil {
		return IncomingMessage{}, err
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return IncomingMessage{}, err
	}
	if strings.TrimSpace(identity.PublicKey) != signerPubkey {
		return IncomingMessage{}, errors.New("message signer does not match local identity")
	}

	if message.Type == "PROFILE_UPDATE" && message.OpID == "" {
		message.OpID = buildMessageID(signerPubkey, fmt.Sprintf("profile|%d", message.Timestamp), message.Timestamp)
	}
	if message.Type == "GOVERNANCE_POLICY_UPDATE" && message.OpID == "" {
		message.OpID = buildMessageID(signerPubkey, fmt.Sprintf("governance-policy|%t|%d", message.HideHistoryOnShadowBan, message.Timestamp), message.Timestamp)
	}
	if (message.Type == "SHADOW_BAN" || message.Type == "UNBAN") && message.OpID == "" {
		message.OpID = buildMessageID(signerPubkey, fmt.Sprintf("governance|%s|%s|%d", message.Type, message.TargetPubkey, message.Timestamp), message.Timestamp)
	}
	if message.Timestamp <= 0 {
		message.Timestamp = time.Now().Unix()
	}

	payload, err := buildIncomingMessageSignaturePayload(message)
	if err != nil {
		return IncomingMessage{}, err
	}
	signature, err := a.SignMessage(identity.Mnemonic, payload)
	if err != nil {
		return IncomingMessage{}, err
	}
	message.Signature = signature
	return message, nil
}

func (a *App) verifyIncomingMessageSignature(message IncomingMessage) error {
	message = normalizeIncomingMessageForSignature(message)
	if !signedIncomingMessageType(message.Type) {
		return nil
	}
	if message.Signature == "" {
		return errors.New("message signature is required")
	}

	signerPubkey, err := signerPubkeyForIncomingMessage(message)
	if err != nil {
		return err
	}
	payload, err := buildIncomingMessageSignaturePayload(message)
	if err != nil {
		return err
	}
	valid, err := a.VerifyMessage(signerPubkey, payload, message.Signature)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid message signature")
	}
	return nil
}
