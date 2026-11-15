import 'package:dio/dio.dart';

import '../data/chatbot_repository.dart';

int _bySort(ChatEntry a, ChatEntry b) => a.sortOrder.compareTo(b.sortOrder);

/// Top-level questions in display order.
List<ChatEntry> topLevelQuestions(List<ChatEntry> all) =>
    all.where((e) => e.parentId == null).toList()..sort(_bySort);

/// Follow-up questions shown after [parentId]'s answer.
List<ChatEntry> followUpQuestions(List<ChatEntry> all, String parentId) =>
    all.where((e) => e.parentId == parentId).toList()..sort(_bySort);

/// Matches a typed [query] against question text (both languages) and admin
/// keywords, case-insensitively. An empty query matches nothing.
List<ChatEntry> searchQuestions(List<ChatEntry> all, String query) {
  final q = query.trim().toLowerCase();
  if (q.isEmpty) return const [];
  return all.where((e) {
    final en = (e.question.values['en'] ?? '').toLowerCase();
    final ar = (e.question.values['ar'] ?? '').toLowerCase();
    return en.contains(q) ||
        ar.contains(q) ||
        e.keywords.any((k) => k.contains(q) || q.contains(k));
  }).toList()..sort(_bySort);
}

/// A wa.me link from a free-typed number ("+961 70 …"), or null.
Uri? whatsappUri(String raw) {
  var digits = raw.replaceAll(RegExp(r'[^0-9]'), '');
  if (digits.startsWith('00')) digits = digits.substring(2);
  return digits.length >= 6 ? Uri.parse('https://wa.me/$digits') : null;
}

/// Which icon a question card shows (the Help Center design gives every card a
/// tinted icon tile; questions carry no icon of their own, so it is inferred).
enum HelpIconKind { cart, wallet, shield, help }

const _cartWords = [
  'buy', 'purchase', 'order', 'product', 'checkout', 'cart', //
  'شراء', 'اشتري', 'طلب', 'منتج',
];
const _walletWords = [
  'wallet', 'top up', 'top-up', 'topup', 'balance', 'deposit', 'fund', //
  'محفظة', 'المحفظة', 'شحن', 'رصيد', 'إيداع',
];
const _shieldWords = [
  'verify', 'verification', 'identity', 'kyc', 'id card', 'passport', //
  'توثيق', 'تحقق', 'هوية', 'الهوية',
];

/// Infers [entry]'s card icon from its question text (EN + AR) and admin
/// keywords. Checked in order: identity → wallet → purchase → generic.
HelpIconKind helpIconKind(ChatEntry entry) {
  final hay = [
    entry.question.values['en'] ?? '',
    entry.question.values['ar'] ?? '',
    ...entry.keywords,
  ].join(' ').toLowerCase();
  bool any(List<String> words) => words.any(hay.contains);
  if (any(_shieldWords)) return HelpIconKind.shield;
  if (any(_walletWords)) return HelpIconKind.wallet;
  if (any(_cartWords)) return HelpIconKind.cart;
  return HelpIconKind.help;
}

/// The grey subtitle under a question card: the answer's first line, cut at
/// its first sentence end when that comes early enough to read as a summary.
String helpSubtitle(String answer) {
  final firstLine = answer
      .trim()
      .split('\n')
      .map((l) => l.trim())
      .firstWhere((l) => l.isNotEmpty, orElse: () => '');
  final end = RegExp(r'[.!?؟。](\s|$)').firstMatch(firstLine);
  if (end != null && end.start >= 20) {
    return firstLine.substring(0, end.start + 1);
  }
  return firstLine;
}

/// Why an AI question got no AI answer.
enum AskFailure {
  /// The customer hit their AI question limit (HTTP 429): say so.
  limited,

  /// The assistant is off, unavailable, or the network failed: fall back to
  /// the keyword match so the customer still gets the quick answers.
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
