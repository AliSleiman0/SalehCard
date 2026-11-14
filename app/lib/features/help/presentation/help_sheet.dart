import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/i18n/arb/app_localizations.dart';
import '../../../core/locale/locale_controller.dart';
import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_tokens.dart';
import '../data/chatbot_repository.dart';
import '../domain/chat_logic.dart';

/// Opens the Help Center (client msg #74) as a bottom sheet. The sheet draws
/// its own rounded chrome + drag handle, so the modal's background is clear.
Future<void> showHelpSheet(BuildContext context) => showModalBottomSheet<void>(
  context: context,
  isScrollControlled: true,
  useSafeArea: true,
  backgroundColor: Colors.transparent,
  builder: (_) =>
      const FractionallySizedBox(heightFactor: 0.9, child: HelpSheet()),
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

const _robotAsset = 'assets/help/robot.png';
const _whatsappGreen = Color(0xFF25D366);

/// Icon + tint per inferred question kind (see [helpIconKind]).
({IconData icon, Color tint}) _kindStyle(HelpIconKind kind) => switch (kind) {
  HelpIconKind.cart => (
    icon: Icons.shopping_cart_outlined,
    tint: const Color(0xFFA56BFF),
  ),
  HelpIconKind.wallet => (
    icon: Icons.account_balance_wallet_outlined,
    tint: const Color(0xFF4F8BFF),
  ),
  HelpIconKind.shield => (
    icon: Icons.verified_user_outlined,
    tint: const Color(0xFF2FD47A),
  ),
  HelpIconKind.help => (
    icon: Icons.help_outline_rounded,
    tint: const Color(0xFFA56BFF),
  ),
};

class _Msg {
  const _Msg(this.fromUser, this.text);
  final bool fromUser;
  final String text;
}

/// The Help Center panel: tap a predefined question card (or type one and
/// send) → its answer appears as a bot bubble, followed by its follow-up
/// questions as cards. WhatsApp + complaint escalation sit at the bottom.
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

  void _scrollToEnd() {
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

  void _ask(ChatEntry e, String locale) {
    setState(() {
      _msgs
        ..add(_Msg(true, e.question.resolve(locale)))
        ..add(_Msg(false, e.answer.resolve(locale)));
      _current = e.parentId ?? e.id;
      _search.clear();
    });
    _scrollToEnd();
  }

  /// Send button / keyboard submit: answers the best match for the typed
  /// question, or says no answer was found.
  void _submit(List<ChatEntry> all, String locale, String noAnswer) {
    final q = _search.text.trim();
    if (q.isEmpty) return;
    final matches = searchQuestions(all, q);
    if (matches.isNotEmpty) {
      _ask(matches.first, locale);
      return;
    }
    setState(() {
      _msgs
        ..add(_Msg(true, q))
        ..add(_Msg(false, noAnswer));
      _search.clear();
    });
    _scrollToEnd();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final locale = ref.watch(localeControllerProvider).languageCode;
    final data = ref.watch(chatbotProvider).value ?? const ChatbotData();
    final all = data.entries;
    final query = _search.text;
    final searching = query.trim().isNotEmpty;
    final suggestions = searching
        ? searchQuestions(all, query)
        : _current != null
        ? followUpQuestions(all, _current!)
        : topLevelQuestions(all);
    final wa = whatsappUri(data.support.whatsapp);
    final keyboard = MediaQuery.viewInsetsOf(context).bottom;
    // Opened from inside the shell, the shell Scaffold already resizes for the
    // keyboard and strips the inset from this context's MediaQuery — so ask
    // the view itself whether the keyboard is up.
    final keyboardOpen = keyboard > 0 || View.of(context).viewInsets.bottom > 0;

    return Container(
      decoration: BoxDecoration(
        color: colors.bg,
        borderRadius: const BorderRadius.vertical(top: Radius.circular(28)),
        border: Border(top: BorderSide(color: colors.borderStrong)),
      ),
      clipBehavior: Clip.antiAlias,
      padding: EdgeInsets.only(bottom: keyboard),
      child: Column(
        children: [
          // Drag handle.
          Container(
            margin: const EdgeInsets.only(top: 10),
            width: 40,
            height: 5,
            decoration: BoxDecoration(
              color: colors.borderStrong,
              borderRadius: BorderRadius.circular(AppTokens.rPill),
            ),
          ),
          _Header(
            title: l10n.helpCenterTitle,
            subtitle: l10n.helpSubtitle,
            onClose: () => Navigator.pop(context),
          ),
          Divider(height: 1, thickness: 1, color: colors.border),
          Expanded(
            child: ListView(
              controller: _scroll,
              padding: const EdgeInsets.fromLTRB(16, 18, 16, 12),
              children: [
                _BotRow(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.helpGreetingTitle,
                        style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w700,
                          color: colors.text,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        l10n.helpGreeting,
                        style: TextStyle(
                          fontSize: 14,
                          height: 1.35,
                          color: Color.lerp(
                            colors.textDim,
                            AppTokens.brandMid,
                            0.25,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                for (final m in _msgs)
                  m.fromUser
                      ? _UserBubble(text: m.text)
                      : _BotRow(
                          child: Text(
                            m.text,
                            style: TextStyle(
                              fontSize: 14,
                              height: 1.4,
                              color: colors.text,
                            ),
                          ),
                        ),
                const SizedBox(height: 6),
                for (final e in suggestions)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: _QuestionCard(
                      entry: e,
                      locale: locale,
                      onTap: () => _ask(e, locale),
                    ),
                  ),
                if (searching && suggestions.isEmpty)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    child: Text(
                      l10n.helpNoMatch,
                      textAlign: TextAlign.center,
                      style: TextStyle(color: colors.textDim, fontSize: 13.5),
                    ),
                  ),
                if (_current != null && !searching)
                  Align(
                    alignment: AlignmentDirectional.centerStart,
                    child: TextButton.icon(
                      onPressed: () => setState(() => _current = null),
                      icon: const Icon(Icons.apps_rounded, size: 18),
                      label: Text(l10n.helpAllQuestions),
                    ),
                  ),
              ],
            ),
          ),
          Padding(
            padding: EdgeInsets.fromLTRB(16, 4, 16, keyboardOpen ? 12 : 4),
            child: _AskInput(
              controller: _search,
              hint: l10n.helpSearchHint,
              sendLabel: l10n.helpSend,
              onChanged: () => setState(() {}),
              onSubmit: () => _submit(all, locale, l10n.helpNoAnswer),
            ),
          ),
          if (!keyboardOpen)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 14, 16, 16),
              child: Column(
                children: [
                  _OrDivider(label: l10n.helpOtherWays),
                  const SizedBox(height: 14),
                  IntrinsicHeight(
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        if (wa != null) ...[
                          Expanded(
                            child: _ContactCard(
                              leading: const _WhatsappGlyph(),
                              accent: _whatsappGreen,
                              title: 'WhatsApp',
                              subtitle: l10n.helpWhatsappSub,
                              onTap: () => launchUrl(
                                wa,
                                mode: LaunchMode.externalApplication,
                              ),
                            ),
                          ),
                          const SizedBox(width: 12),
                        ],
                        // Escalate to a tracked complaint the support team
                        // follows up on.
                        Expanded(
                          child: _ContactCard(
                            leading: const Icon(
                              Icons.headset_mic_outlined,
                              color: Color(0xFFB98CFF),
                              size: 30,
                            ),
                            title: l10n.complaintSubmit,
                            subtitle: l10n.helpReportIssue,
                            onTap: () {
                              final router = GoRouter.of(context);
                              Navigator.pop(context);
                              router.push('/complaints/new');
                            },
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

/// Gradient chat tile, title + subtitle, round close button, glowing wave.
class _Header extends StatelessWidget {
  const _Header({
    required this.title,
    required this.subtitle,
    required this.onClose,
  });

  final String title;
  final String subtitle;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final rtl = Directionality.of(context) == TextDirection.rtl;
    return Stack(
      children: [
        Positioned.fill(
          child: IgnorePointer(
            child: CustomPaint(painter: _WavePainter(rtl: rtl)),
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 14, 14, 22),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(16),
                  gradient: const LinearGradient(
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                    colors: [AppTokens.brandMid, AppTokens.brand2],
                  ),
                  boxShadow: [
                    BoxShadow(
                      color: AppTokens.brandMid.withValues(alpha: 0.45),
                      blurRadius: 18,
                      offset: const Offset(0, 6),
                    ),
                  ],
                ),
                child: const Icon(
                  Icons.chat_rounded,
                  color: Colors.white,
                  size: 28,
                ),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        style: TextStyle(
                          fontSize: 22,
                          fontWeight: FontWeight.w800,
                          color: colors.text,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        subtitle,
                        style: TextStyle(fontSize: 14, color: colors.textDim),
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(width: 8),
              Material(
                color: colors.surface,
                shape: CircleBorder(side: BorderSide(color: colors.border)),
                child: InkWell(
                  customBorder: const CircleBorder(),
                  onTap: onClose,
                  child: SizedBox(
                    width: 40,
                    height: 40,
                    child: Icon(
                      Icons.close_rounded,
                      size: 20,
                      color: colors.text,
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

/// The glowing violet → magenta wave along the header's bottom-end corner.
class _WavePainter extends CustomPainter {
  const _WavePainter({required this.rtl});

  final bool rtl;

  @override
  void paint(Canvas canvas, Size size) {
    if (rtl) {
      canvas
        ..translate(size.width, 0)
        ..scale(-1, 1);
    }
    final w = size.width;
    final h = size.height;
    final path = Path()
      ..moveTo(w * 0.28, h * 1.0)
      ..cubicTo(w * 0.55, h * 1.0, w * 0.70, h * 0.90, w * 0.84, h * 0.86)
      ..cubicTo(w * 0.93, h * 0.83, w * 0.97, h * 0.68, w * 1.02, h * 0.62);
    final shader = const LinearGradient(
      colors: [Color(0x008A3BFF), AppTokens.brandMid, AppTokens.brand2],
      stops: [0, 0.55, 1],
    ).createShader(Rect.fromLTWH(w * 0.30, 0, w * 0.72, h));
    final glow = Paint()
      ..shader = shader
      ..style = PaintingStyle.stroke
      ..strokeWidth = 7
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 8);
    final line = Paint()
      ..shader = shader
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.round
      ..strokeWidth = 2;
    canvas
      ..drawPath(path, glow)
      ..drawPath(path, line);
  }

  @override
  bool shouldRepaint(_WavePainter old) => old.rtl != rtl;
}

/// The robot avatar with a soft violet glow behind it.
class _RobotAvatar extends StatelessWidget {
  const _RobotAvatar();

  @override
  Widget build(BuildContext context) => Container(
    width: 48,
    height: 48,
    decoration: BoxDecoration(
      shape: BoxShape.circle,
      gradient: RadialGradient(
        colors: [
          AppTokens.brandMid.withValues(alpha: 0.55),
          AppTokens.brandMid.withValues(alpha: 0),
        ],
      ),
    ),
    padding: const EdgeInsets.all(3),
    child: Image.asset(_robotAsset, filterQuality: FilterQuality.medium),
  );
}

/// A bot message: avatar + bubble with a tail pointing at it.
class _BotRow extends StatelessWidget {
  const _BotRow({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final rtl = Directionality.of(context) == TextDirection.rtl;
    return Padding(
      padding: const EdgeInsets.only(bottom: 14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const _RobotAvatar(),
          const SizedBox(width: 12),
          Flexible(
            child: Stack(
              clipBehavior: Clip.none,
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 13,
                  ),
                  decoration: BoxDecoration(
                    color: colors.surface,
                    border: Border.all(color: colors.border),
                    borderRadius: BorderRadius.circular(18),
                  ),
                  child: child,
                ),
                PositionedDirectional(
                  start: -7,
                  top: 16,
                  child: CustomPaint(
                    size: const Size(9, 14),
                    painter: _TailPainter(
                      fill: colors.surface,
                      stroke: colors.border,
                      rtl: rtl,
                    ),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 24),
        ],
      ),
    );
  }
}

/// The bubble's little speech tail; overlaps the bubble's border by 1px so the
/// border line under it is hidden.
class _TailPainter extends CustomPainter {
  const _TailPainter({
    required this.fill,
    required this.stroke,
    required this.rtl,
  });

  final Color fill;
  final Color stroke;
  final bool rtl;

  @override
  void paint(Canvas canvas, Size size) {
    if (rtl) {
      canvas
        ..translate(size.width, 0)
        ..scale(-1, 1);
    }
    final w = size.width;
    final h = size.height;
    canvas.drawPath(
      Path()
        ..moveTo(w + 1, 0)
        ..lineTo(0, h / 2)
        ..lineTo(w + 1, h)
        ..close(),
      Paint()..color = fill,
    );
    canvas.drawPath(
      Path()
        ..moveTo(w, 0)
        ..lineTo(0, h / 2)
        ..lineTo(w, h),
      Paint()
        ..color = stroke
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1,
    );
  }

  @override
  bool shouldRepaint(_TailPainter old) =>
      old.fill != fill || old.stroke != stroke || old.rtl != rtl;
}

/// The customer's question, end-aligned in the brand gradient.
class _UserBubble extends StatelessWidget {
  const _UserBubble({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) => Align(
    alignment: AlignmentDirectional.centerEnd,
    child: Container(
      margin: const EdgeInsetsDirectional.only(start: 60, bottom: 14),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          colors: [AppTokens.brandMid, AppTokens.brand2],
        ),
        borderRadius: BorderRadiusDirectional.only(
          topStart: Radius.circular(18),
          topEnd: Radius.circular(18),
          bottomStart: Radius.circular(18),
          bottomEnd: Radius.circular(6),
        ),
      ),
      child: Text(
        text,
        style: const TextStyle(
          fontSize: 14,
          height: 1.4,
          color: Colors.white,
          fontWeight: FontWeight.w600,
        ),
      ),
    ),
  );
}

/// A tappable question: tinted icon tile, title, one-line answer summary,
/// chevron.
class _QuestionCard extends StatelessWidget {
  const _QuestionCard({
    required this.entry,
    required this.locale,
    required this.onTap,
  });

  final ChatEntry entry;
  final String locale;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final style = _kindStyle(helpIconKind(entry));
    final subtitle = helpSubtitle(entry.answer.resolve(locale));
    return Material(
      color: colors.surface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(18),
        side: BorderSide(color: colors.border),
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(18),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsetsDirectional.fromSTEB(14, 14, 10, 14),
          child: Row(
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  color: style.tint.withValues(alpha: 0.16),
                  borderRadius: BorderRadius.circular(14),
                ),
                child: Icon(style.icon, color: style.tint, size: 26),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      entry.question.resolve(locale),
                      style: TextStyle(
                        fontSize: 16.5,
                        fontWeight: FontWeight.w700,
                        color: colors.text,
                      ),
                    ),
                    if (subtitle.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(
                        subtitle,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 13.5,
                          height: 1.35,
                          color: colors.textDim,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              const SizedBox(width: 6),
              // Mirrors itself in RTL.
              Icon(Icons.chevron_right_rounded, color: colors.textDim),
            ],
          ),
        ),
      ),
    );
  }
}

/// The pill "Type your question..." field with a gradient send button.
class _AskInput extends StatelessWidget {
  const _AskInput({
    required this.controller,
    required this.hint,
    required this.sendLabel,
    required this.onChanged,
    required this.onSubmit,
  });

  final TextEditingController controller;
  final String hint;
  final String sendLabel;
  final VoidCallback onChanged;
  final VoidCallback onSubmit;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      height: 58,
      padding: const EdgeInsetsDirectional.only(start: 16, end: 6),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(AppTokens.rPill),
        border: Border.all(color: colors.borderStrong),
      ),
      child: Row(
        children: [
          Icon(Icons.search_rounded, color: colors.textDim, size: 22),
          const SizedBox(width: 10),
          Expanded(
            child: TextField(
              controller: controller,
              onChanged: (_) => onChanged(),
              onSubmitted: (_) => onSubmit(),
              textInputAction: TextInputAction.send,
              style: TextStyle(fontSize: 15, color: colors.text),
              decoration: InputDecoration(
                hintText: hint,
                hintStyle: TextStyle(color: colors.textDim, fontSize: 15),
                border: InputBorder.none,
                enabledBorder: InputBorder.none,
                focusedBorder: InputBorder.none,
                filled: false,
                isCollapsed: true,
              ),
            ),
          ),
          const SizedBox(width: 8),
          Semantics(
            button: true,
            label: sendLabel,
            child: InkWell(
              customBorder: const CircleBorder(),
              onTap: onSubmit,
              child: Container(
                width: 46,
                height: 46,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  gradient: const LinearGradient(
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                    colors: [AppTokens.brandMid, AppTokens.brand2],
                  ),
                  boxShadow: [
                    BoxShadow(
                      color: AppTokens.brandMid.withValues(alpha: 0.4),
                      blurRadius: 12,
                      offset: const Offset(0, 4),
                    ),
                  ],
                ),
                // send_rounded already mirrors itself in RTL.
                child: const Icon(
                  Icons.send_rounded,
                  color: Colors.white,
                  size: 20,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// "—— Other ways to get help ——".
class _OrDivider extends StatelessWidget {
  const _OrDivider({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final line = Expanded(child: Divider(color: colors.border, thickness: 1));
    return Row(
      children: [
        line,
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12),
          child: Text(
            label,
            style: TextStyle(fontSize: 13.5, color: colors.textDim),
          ),
        ),
        line,
      ],
    );
  }
}

/// A bottom escalation card (WhatsApp / Submit a complaint).
class _ContactCard extends StatelessWidget {
  const _ContactCard({
    required this.leading,
    required this.title,
    required this.subtitle,
    required this.onTap,
    this.accent,
  });

  final Widget leading;
  final Color? accent;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Material(
      color: accent?.withValues(alpha: 0.06) ?? colors.surface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(
          color: accent?.withValues(alpha: 0.55) ?? colors.borderStrong,
        ),
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
          child: Row(
            children: [
              leading,
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      title,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 14,
                        height: 1.2,
                        fontWeight: FontWeight.w700,
                        color: colors.text,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      subtitle,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(fontSize: 12.5, color: colors.textDim),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// A WhatsApp-style mark (ring with a speech tail around a handset), drawn
/// so the app needs no icon-font/SVG dependency for one glyph.
class _WhatsappGlyph extends StatelessWidget {
  const _WhatsappGlyph();

  @override
  Widget build(BuildContext context) => const SizedBox(
    width: 30,
    height: 30,
    child: CustomPaint(
      painter: _WhatsappRingPainter(),
      child: Center(
        child: Icon(Icons.call_rounded, size: 15, color: _whatsappGreen),
      ),
    ),
  );
}

class _WhatsappRingPainter extends CustomPainter {
  const _WhatsappRingPainter();

  @override
  void paint(Canvas canvas, Size size) {
    final c = size.center(Offset.zero);
    final r = size.width / 2 - 2;
    final paint = Paint()
      ..color = _whatsappGreen
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.4
      ..strokeJoin = StrokeJoin.round;
    // Ring open at the bottom-left, closed by the speech tail.
    final path = Path()
      ..addArc(Rect.fromCircle(center: c, radius: r), 2.25, 5.6)
      ..lineTo(c.dx - r * 0.95, c.dy + r * 1.02)
      ..close();
    canvas.drawPath(path, paint);
  }

  @override
  bool shouldRepaint(_WhatsappRingPainter old) => false;
}
