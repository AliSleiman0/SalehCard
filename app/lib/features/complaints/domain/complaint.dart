/// Lifecycle of a customer complaint. A customer reply reopens a resolved
/// complaint; a closed one accepts no more replies.
enum ComplaintStatus { open, inProgress, resolved, closed, unknown }

ComplaintStatus complaintStatusFromString(String raw) {
  switch (raw) {
    case 'open':
      return ComplaintStatus.open;
    case 'in_progress':
      return ComplaintStatus.inProgress;
    case 'resolved':
      return ComplaintStatus.resolved;
    case 'closed':
      return ComplaintStatus.closed;
    default:
      return ComplaintStatus.unknown;
  }
}

/// What the complaint is about (server keys).
const complaintCategories = ['order', 'topup', 'account', 'other'];

/// One message in a complaint thread. Staff internal notes never reach the
/// app (the server strips them).
class ComplaintMessage {
  const ComplaintMessage({
    required this.id,
    required this.fromCustomer,
    required this.body,
    required this.createdAt,
    this.photoUrl,
  });

  final String id;
  final bool fromCustomer;
  final String body;
  final String? photoUrl;
  final DateTime createdAt;

  factory ComplaintMessage.fromJson(Map<String, dynamic> json) =>
      ComplaintMessage(
        id: json['id'] as String? ?? '',
        fromCustomer: (json['author'] as String? ?? '') == 'customer',
        body: json['body'] as String? ?? '',
        photoUrl: (json['photoUrl'] as String?)?.isEmpty ?? true
            ? null
            : json['photoUrl'] as String,
        createdAt:
            DateTime.tryParse(json['createdAt'] as String? ?? '') ??
            DateTime.fromMillisecondsSinceEpoch(0),
      );
}

/// A customer complaint. [messages] is empty on list rows.
class Complaint {
  const Complaint({
    required this.id,
    required this.ref,
    required this.category,
    required this.subject,
    required this.status,
    required this.updatedAt,
    this.orderId,
    this.messages = const [],
  });

  final String id;
  final String ref;
  final String category;
  final String subject;
  final ComplaintStatus status;
  final String? orderId;
  final List<ComplaintMessage> messages;
  final DateTime updatedAt;

  bool get isClosed => status == ComplaintStatus.closed;

  factory Complaint.fromJson(Map<String, dynamic> json) => Complaint(
    id: json['id'] as String? ?? '',
    ref: json['ref'] as String? ?? '',
    category: json['category'] as String? ?? 'other',
    subject: json['subject'] as String? ?? '',
    status: complaintStatusFromString(json['status'] as String? ?? ''),
    orderId: json['orderId'] as String?,
    messages: [
      for (final m in (json['messages'] as List<dynamic>? ?? const []))
        ComplaintMessage.fromJson(m as Map<String, dynamic>),
    ],
    updatedAt:
        DateTime.tryParse(json['updatedAt'] as String? ?? '') ??
        DateTime.fromMillisecondsSinceEpoch(0),
  );
}

/// Notification kinds the complaint desk sends.
const complaintNotificationKinds = {'complaint_reply', 'complaint_status'};

/// The app route a complaint notification (push payload or inbox row) should
/// open, or null when [kind]/[data] are not a complaint notification.
String? complaintRouteFor(String? kind, Map<String, dynamic> data) {
  if (kind == null || !complaintNotificationKinds.contains(kind)) return null;
  final id = data['complaintId']?.toString() ?? '';
  // Ids are 24-char hex ObjectIDs; anything else is not routable.
  if (!RegExp(r'^[0-9a-fA-F]{24}$').hasMatch(id)) return null;
  return '/complaints/$id';
}
