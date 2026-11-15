import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/config/regulated_finance.dart';
import '../../../core/i18n/i18n_string.dart';
import '../../../core/network/api_envelope.dart';
import '../../../core/network/providers.dart';

/// One predefined help-bot question and answer (admin-managed quick question).
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
    this.ai = false,
  });

  final List<ChatEntry> entries;
  final ChatSupport support;

  /// Whether typed questions go to the AI assistant (admin Settings toggle +
  /// a configured provider). Off → the keyword match below is used.
  final bool ai;

  factory ChatbotData.fromJson(Map<String, dynamic> json) => ChatbotData(
    entries: [
      for (final e in (json['entries'] as List<dynamic>? ?? const []))
        ChatEntry.fromJson(e as Map<String, dynamic>),
    ],
    support: ChatSupport.fromJson(json['support'] as Map<String, dynamic>?),
    ai: json['ai'] as bool? ?? false,
  );
}

/// The AI assistant's reply. [answered] false = [answer] is the server's
/// polite "can't help with that" text and the support contact should show.
class AskResult {
  const AskResult({
    required this.answer,
    required this.answered,
    this.logId = '',
    this.showSupport = false,
  });

  final String answer;
  final bool answered;
  final String logId;
  final bool showSupport;

  factory AskResult.fromJson(Map<String, dynamic> json) => AskResult(
    answer: json['answer'] as String? ?? '',
    answered: json['answered'] as bool? ?? false,
    logId: json['logId'] as String? ?? '',
    showSupport: json['showSupport'] as bool? ?? false,
  );
}

/// One earlier turn sent along for context ("user" | "assistant").
class AskTurn {
  const AskTurn(this.role, this.content);
  final String role;
  final String content;
  Map<String, dynamic> toJson() => {'role': role, 'content': content};
}

/// Talks to the AI assistant endpoints (authenticated).
class ChatbotRepository {
  ChatbotRepository(this._dio);
  final Dio _dio;

  /// POST /chatbot/ask. Throws [DioException] on non-2xx (429 = the
  /// customer's limit, 503 = assistant off/unavailable) — see [askFailure].
  Future<AskResult> ask(
    String question, {
    required String lang,
    List<AskTurn> history = const [],
  }) async {
    final response = await _dio.post<dynamic>(
      '/chatbot/ask',
      data: {
        'question': question,
        'lang': lang,
        // iOS must never be answered from android-only articles (App Review);
        // keyed off the same build scope that hides those features.
        'platform': restrictToRealWorldServices ? 'ios' : 'android',
        'history': [for (final t in history) t.toJson()],
      },
    );
    return AskResult.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// POST /chatbot/ask/{logId}/report — flags an AI answer for review.
  Future<void> report(String logId) async {
    await _dio.post<dynamic>('/chatbot/ask/$logId/report');
  }
}

final chatbotRepositoryProvider = Provider<ChatbotRepository>(
  (ref) => ChatbotRepository(ref.watch(dioProvider)),
);

/// GET /chatbot — public; an outage just hides the help button.
final chatbotProvider = FutureProvider<ChatbotData>((ref) async {
  final response = await ref.watch(dioProvider).get<dynamic>('/chatbot');
  return ChatbotData.fromJson(unwrap(response) as Map<String, dynamic>);
});
