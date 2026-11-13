import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/i18n/i18n_string.dart';
import '../../../core/network/api_envelope.dart';
import '../../../core/network/providers.dart';

/// One predefined help-bot question and answer (admin-managed; no AI).
/// [parentId] set = a follow-up shown after its parent's answer.
class ChatEntry {
  const ChatEntry({
    required this.id,
    required this.question,
    required this.answer,
    this.parentId,
    this.keywords = const [],
    this.sortOrder = 0,
  });

  final String id;
  final String? parentId;
  final I18nString question;
  final I18nString answer;
  final List<String> keywords;
  final int sortOrder;

  factory ChatEntry.fromJson(Map<String, dynamic> json) => ChatEntry(
    id: json['id'] as String? ?? '',
    parentId: json['parentId'] as String?,
    question: I18nString.fromJson(json['question']),
    answer: I18nString.fromJson(json['answer']),
    keywords: [
      for (final k in (json['keywords'] as List<dynamic>? ?? const []))
        k.toString(),
    ],
    sortOrder: (json['sortOrder'] as num?)?.toInt() ?? 0,
  );
}

/// The "still need help?" contact from the admin Settings.
class ChatSupport {
  const ChatSupport({this.email = '', this.phone = '', this.whatsapp = ''});

  final String email;
  final String phone;
  final String whatsapp;

  bool get isEmpty => email.isEmpty && phone.isEmpty && whatsapp.isEmpty;

  factory ChatSupport.fromJson(Map<String, dynamic>? json) => ChatSupport(
    email: json?['email'] as String? ?? '',
    phone: json?['phone'] as String? ?? '',
    whatsapp: json?['whatsapp'] as String? ?? '',
  );
}

class ChatbotData {
  const ChatbotData({
    this.entries = const [],
    this.support = const ChatSupport(),
  });

  final List<ChatEntry> entries;
  final ChatSupport support;

  factory ChatbotData.fromJson(Map<String, dynamic> json) => ChatbotData(
    entries: [
      for (final e in (json['entries'] as List<dynamic>? ?? const []))
        ChatEntry.fromJson(e as Map<String, dynamic>),
    ],
    support: ChatSupport.fromJson(json['support'] as Map<String, dynamic>?),
  );
}

/// GET /chatbot — public; an outage just hides the help button.
final chatbotProvider = FutureProvider<ChatbotData>((ref) async {
  final response = await ref.watch(dioProvider).get<dynamic>('/chatbot');
  return ChatbotData.fromJson(unwrap(response) as Map<String, dynamic>);
});
