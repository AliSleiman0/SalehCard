import 'package:dio/dio.dart';

import '../data/chatbot_repository.dart';

/// Whether the Help Center has anything to offer: the AI assistant, or a
/// WhatsApp contact (the complaint card alone is reachable from the menu).
bool showHelpCenter(ChatbotData d) =>
    d.ai || whatsappUri(d.support.whatsapp) != null;

/// A wa.me link from a free-typed number ("+961 70 …"), or null.
Uri? whatsappUri(String raw) {
  var digits = raw.replaceAll(RegExp(r'[^0-9]'), '');
  if (digits.startsWith('00')) digits = digits.substring(2);
  return digits.length >= 6 ? Uri.parse('https://wa.me/$digits') : null;
}

/// Why an AI question got no AI answer.
enum AskFailure {
  /// The customer hit their AI question limit (HTTP 429): say so.
  limited,

  /// The assistant is off, unavailable, or the network failed: say so and
  /// point the customer at the support contacts.
  unavailable,
}

/// Maps an error from [ChatbotRepository.ask] to an [AskFailure].
AskFailure askFailure(Object error) {
  if (error is DioException && error.response?.statusCode == 429) {
    return AskFailure.limited;
  }
  return AskFailure.unavailable;
}

/// The last [maxTurns] conversation turns to send as context, oldest first.
/// [turns] are (fromUser, text) pairs in display order; pending placeholders
/// must already be excluded.
List<AskTurn> askHistory(
  List<({bool fromUser, String text})> turns, {
  int maxTurns = 6,
}) {
  final start = turns.length > maxTurns ? turns.length - maxTurns : 0;
  return [
    for (final t in turns.sublist(start))
      AskTurn(t.fromUser ? 'user' : 'assistant', t.text),
  ];
}
