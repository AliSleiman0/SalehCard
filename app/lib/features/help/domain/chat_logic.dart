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
