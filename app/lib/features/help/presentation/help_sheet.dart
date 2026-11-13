import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/i18n/arb/app_localizations.dart';
import '../../../core/locale/locale_controller.dart';
import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_tokens.dart';
import '../data/chatbot_repository.dart';
import '../domain/chat_logic.dart';

/// Opens the help bot (client msg #74) as a bottom sheet.
Future<void> showHelpSheet(BuildContext context) => showModalBottomSheet<void>(
  context: context,
  isScrollControlled: true,
  useSafeArea: true,
  builder: (_) =>
      const FractionallySizedBox(heightFactor: 0.85, child: HelpSheet()),
);

/// Floating help button; hidden while the admin has added no questions (or
/// the chatbot endpoint is unreachable).
class HelpFab extends ConsumerWidget {
  const HelpFab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final has = ref
        .watch(chatbotProvider)
        .maybeWhen(data: (d) => d.entries.isNotEmpty, orElse: () => false);
    if (!has) return const SizedBox.shrink();
    return FloatingActionButton(
      heroTag: 'help-fab',
      tooltip: AppLocalizations.of(context).helpTitle,
      onPressed: () => showHelpSheet(context),
      child: const Icon(Icons.chat_bubble_outline_rounded),
    );
  }
}

class _Msg {
  const _Msg(this.fromUser, this.text);
  final bool fromUser;
  final String text;
}

/// The chat-style panel: tap a predefined question → its answer appears as a
/// bot bubble, followed by its follow-up questions as chips.
class HelpSheet extends ConsumerStatefulWidget {
  const HelpSheet({super.key});

  @override
  ConsumerState<HelpSheet> createState() => _HelpSheetState();
}

class _HelpSheetState extends ConsumerState<HelpSheet> {
  final _msgs = <_Msg>[];
  final _scroll = ScrollController();
  final _search = TextEditingController();
  String? _current; // top-level id whose follow-ups are offered

  @override
  void dispose() {
    _scroll.dispose();
    _search.dispose();
    super.dispose();
  }

  void _ask(ChatEntry e, String locale) {
    setState(() {
      _msgs
        ..add(_Msg(true, e.question.resolve(locale)))
        ..add(_Msg(false, e.answer.resolve(locale)));
      _current = e.parentId ?? e.id;
      _search.clear();
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(
          _scroll.position.maxScrollExtent,
          duration: const Duration(milliseconds: 250),
          curve: Curves.easeOut,
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final locale = ref.watch(localeControllerProvider).languageCode;
    final data = ref.watch(chatbotProvider).value ?? const ChatbotData();
    final all = data.entries;
    final query = _search.text;
    final suggestions = query.trim().isNotEmpty
        ? searchQuestions(all, query)
        : _current != null
        ? followUpQuestions(all, _current!)
        : topLevelQuestions(all);
    final wa = whatsappUri(data.support.whatsapp);

    Widget bubble(_Msg m) => Align(
      alignment: m.fromUser
          ? AlignmentDirectional.centerEnd
          : AlignmentDirectional.centerStart,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
        constraints: const BoxConstraints(maxWidth: 300),
        decoration: BoxDecoration(
          color: m.fromUser ? AppTokens.brand1 : colors.surface,
          border: m.fromUser ? null : Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(14),
        ),
        child: Text(
          m.text,
          style: TextStyle(
            fontSize: 14,
            height: 1.4,
            color: m.fromUser ? Colors.white : colors.text,
          ),
        ),
      ),
    );

    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(18, 14, 8, 6),
            child: Row(
              children: [
                const Icon(Icons.chat_bubble_outline_rounded, size: 20),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    l10n.helpTitle,
                    style: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                ),
                IconButton(
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(Icons.close),
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(
            child: ListView(
              controller: _scroll,
              padding: const EdgeInsets.all(16),
              children: [
                bubble(_Msg(false, l10n.helpGreeting)),
                for (final m in _msgs) bubble(m),
              ],
            ),
          ),
          const Divider(height: 1),
          ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 160),
            child: SingleChildScrollView(
              padding: const EdgeInsets.fromLTRB(16, 10, 16, 6),
              child: Wrap(
                spacing: 6,
                runSpacing: 6,
                children: [
                  for (final e in suggestions)
                    ActionChip(
                      label: Text(e.question.resolve(locale)),
                      onPressed: () => _ask(e, locale),
                    ),
                  if (query.trim().isNotEmpty && suggestions.isEmpty)
                    Text(
                      l10n.helpNoMatch,
                      style: TextStyle(color: colors.textDim),
                    ),
                  if (_current != null && query.trim().isEmpty)
                    ActionChip(
                      label: Text(l10n.helpAllQuestions),
                      onPressed: () => setState(() => _current = null),
                    ),
                ],
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
            child: TextField(
              controller: _search,
              onChanged: (_) => setState(() {}),
              decoration: InputDecoration(
                isDense: true,
                hintText: l10n.helpSearchHint,
                prefixIcon: const Icon(Icons.search, size: 20),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(AppTokens.rMd),
                ),
              ),
            ),
          ),
          if (!data.support.isEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 14),
              child: Wrap(
                crossAxisAlignment: WrapCrossAlignment.center,
                spacing: 12,
                children: [
                  Text(
                    l10n.helpNeedHelp,
                    style: TextStyle(color: colors.textDim, fontSize: 12.5),
                  ),
                  if (wa != null)
                    TextButton(
                      onPressed: () =>
                          launchUrl(wa, mode: LaunchMode.externalApplication),
                      child: const Text('WhatsApp'),
                    ),
                  if (data.support.phone.isNotEmpty)
                    TextButton(
                      onPressed: () => launchUrl(
                        Uri(
                          scheme: 'tel',
                          path: data.support.phone.replaceAll(' ', ''),
                        ),
                      ),
                      child: Text(
                        data.support.phone,
                        textDirection: TextDirection.ltr,
                      ),
                    ),
                  if (data.support.email.isNotEmpty)
                    TextButton(
                      onPressed: () => launchUrl(
                        Uri(scheme: 'mailto', path: data.support.email),
                      ),
                      child: Text(data.support.email),
                    ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}
