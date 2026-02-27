import re

file_path = 'aegis-app/p2p.go'

with open(file_path, 'r') as f:
    content = f.read()

# Add REPORT message type
if 'messageTypePostFetchResponse = "POST_FETCH_RESPONSE"' in content:
    content = content.replace('messageTypePostFetchResponse = "POST_FETCH_RESPONSE"', 'messageTypePostFetchResponse = "POST_FETCH_RESPONSE"\n\tmessageTypeReport            = "REPORT"')
else:
    print("Could not find message type constants")

# Add handling in handleStream
if 'case messageTypePostFetchResponse:' in content:
    new_case = """	case messageTypeReport:
		if err := app.handleReport(msg); err != nil {
			log.Printf("Failed to handle report from %s: %v", peerID, err)
		}
"""
    content = content.replace('case messageTypePostFetchResponse:', new_case + '\n\tcase messageTypePostFetchResponse:')
else:
    print("Could not find handleStream switch")

# Add handleReport method
report_method = """
func (a *App) handleReport(msg P2PMessage) error {
    var report Report
    if err := json.Unmarshal(msg.Payload, &report); err != nil {
        return fmt.Errorf("invalid report payload: %w", err)
    }

    // Verify signature or origin if possible (omitted for now as Report struct doesn't have signature yet,
    // assuming trust or will add signature later. For MVP, we trust the message integrity).

    // Check if we are an admin or if this node cares about reports.
    // For now, we store all valid reports we see if we are configured as an admin node,
    // OR we just store them to allow this node to act as a moderator.

    // To prevent spam, we might want to check if the reporter is "trusted" or if the PoW is valid.

    return a.SubmitReport(report.TargetID, report.TargetType, report.Reason)
}

func (a *App) BroadcastReport(report Report) error {
    payload, err := json.Marshal(report)
    if err != nil {
        return err
    }

    msg := P2PMessage{
        Type:      messageTypeReport,
        Payload:   payload,
        Timestamp: time.Now().Unix(),
    }

    return a.broadcast(msg)
}
"""

content += "\n" + report_method

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
