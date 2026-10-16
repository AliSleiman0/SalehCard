import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../domain/entities/order.dart';
import '../providers.dart';

/// Order success / processing screen. Uses the [order] passed via router `extra`
/// when available (avoids a refetch right after placing it); otherwise loads it
/// by [id]. Delivered codes show per line with a copy button (also on a mixed
/// order that is still processing its manual lines); account_credit / transfer
/// orders show the processing variant.
class OrderSuccessScreen extends ConsumerWidget {
  const OrderSuccessScreen({super.key, required this.id, this.order});

  final String id;
  final Order? order;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;

    if (order != null) {
      return _Scaffold(order: order!);
    }

    final orderAsync = ref.watch(orderDetailProvider(id));
    return Scaffold(
      backgroundColor: colors.bg,
      appBar: _appBar(context, l10n, colors),
      body: orderAsync.when(
        loading: () => const LoadingView(),
        error: (_, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.loadFailed, textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () => context.go('/home'),
                  child: Text(l10n.backToHomeCta),
                ),
              ],
            ),
          ),
        ),
        data: (data) => _SuccessBody(order: data),
      ),
    );
  }

  static PreferredSizeWidget _appBar(
      BuildContext context, AppLocalizations l10n, AppColors colors) {
    return AppBar(
      backgroundColor: colors.topbar,
      automaticallyImplyLeading: false,
      leading: IconButton(
        icon: const Icon(Icons.close_rounded),
        onPressed: () => context.go('/home'),
      ),
    );
  }
}

class _Scaffold extends ConsumerWidget {
  const _Scaffold({required this.order});

  final Order order;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    return Scaffold(
      backgroundColor: colors.bg,
      appBar: OrderSuccessScreen._appBar(context, l10n, colors),
      body: _SuccessBody(order: order),
    );
  }
}

class _SuccessBody extends ConsumerStatefulWidget {
  const _SuccessBody({required this.order});

  final Order order;

  @override
  ConsumerState<_SuccessBody> createState() => _SuccessBodyState();
}

class _SuccessBodyState extends ConsumerState<_SuccessBody> {
  /// The code most recently copied (drives that row's "Copied" pill).
  String? _copiedCode;

  Future<void> _copy(String code) async {
    await Clipboard.setData(ClipboardData(text: code));
    if (!mounted) return;
    setState(() => _copiedCode = code);
    Future.delayed(const Duration(milliseconds: 1600), () {
      if (mounted && _copiedCode == code) setState(() => _copiedCode = null);
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final order = widget.order;
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final codeLines = order.codeLines;
    final processing = order.isProcessing || codeLines.isEmpty;
    // Codes delivered but manual (credit/transfer) lines still pending.
    final partial =
        codeLines.isNotEmpty && order.isProcessing && order.hasManualLines;

    return ListView(
      padding: const EdgeInsets.fromLTRB(24, 16, 24, 28),
      children: [
        Center(
          child: Column(
            children: [
              _StatusCircle(processing: processing),
              const SizedBox(height: 22),
              Text(
                processing
                    ? l10n.orderProcessingTitle
                    : l10n.orderCompletedTitle,
                textAlign: TextAlign.center,
                style: TextStyle(
                    fontSize: 24,
                    fontWeight: FontWeight.w800,
                    color: colors.text),
              ),
              const SizedBox(height: 9),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: Text(
                  processing
                      ? l10n.orderProcessingSub
                      : l10n.orderCompletedSub,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                      fontSize: 14.5, height: 1.5, color: colors.textDim),
                ),
              ),
              const SizedBox(height: 14),
              Text.rich(
                TextSpan(
                  text: '${l10n.orderLabel} ',
                  style: TextStyle(
                      fontSize: 12.5,
                      fontWeight: FontWeight.w600,
                      letterSpacing: 0.3,
                      color: colors.textFaint),
                  children: [
                    TextSpan(
                      text: '#${order.reference}',
                      style: TextStyle(
                          fontFeatures: const [FontFeature.tabularFigures()],
                          fontWeight: FontWeight.w700,
                          color: colors.text),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 24),
        if (codeLines.isNotEmpty) ...[
          Text(l10n.deliveredCodesLabel,
              style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w800,
                  color: colors.text)),
          const SizedBox(height: 11),
          for (var i = 0; i < codeLines.length; i++) ...[
            if (i > 0) const SizedBox(height: 10),
            _CodeCard(
              label: _codeLabel(codeLines[i], localeCode, l10n),
              codes: codeLines[i].codes,
              copiedCode: _copiedCode,
              onCopy: _copy,
              l10n: l10n,
            ),
          ],
          if (partial) ...[
            const SizedBox(height: 14),
            _ProcessingNote(text: l10n.orderRemainingProcessingNote),
          ],
        ] else
          _ProcessingNote(text: l10n.orderProcessingNote),
        const SizedBox(height: 26),
        OutlinedButton(
          onPressed: () => context.go('/orders/${order.id}'),
          style: OutlinedButton.styleFrom(
            minimumSize: const Size.fromHeight(54),
            foregroundColor: colors.text,
            side: BorderSide(color: colors.borderStrong),
            shape: const StadiumBorder(),
            textStyle:
                const TextStyle(fontSize: 16, fontWeight: FontWeight.w800),
          ),
          child: Text(l10n.viewOrderCta),
        ),
        const SizedBox(height: 11),
        FilledButton(
          onPressed: () => context.go('/home'),
          style: FilledButton.styleFrom(
            minimumSize: const Size.fromHeight(54),
            backgroundColor: AppTokens.cta,
            foregroundColor: Colors.white,
            shape: const StadiumBorder(),
            elevation: 8,
            shadowColor: AppTokens.cta.withValues(alpha: 0.3),
            textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w800),
          ),
          child: Text(l10n.backToHomeCta),
        ),
      ],
    );
  }

  String _codeLabel(
      OrderCodeLine line, String localeCode, AppLocalizations l10n) {
    final label = line.label(localeCode);
    return label.isEmpty ? l10n.giftCardLabel : label;
  }
}

class _StatusCircle extends StatelessWidget {
  const _StatusCircle({required this.processing});

  final bool processing;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 92,
      height: 92,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        gradient: AppTokens.brandGradient,
        boxShadow: [
          BoxShadow(
            color: AppTokens.brandMid.withValues(alpha: 0.55),
            blurRadius: 34,
            offset: const Offset(0, 14),
          ),
        ],
      ),
      child: Icon(
        processing ? Icons.schedule_rounded : Icons.check_rounded,
        size: processing ? 42 : 46,
        color: Colors.white,
      ),
    );
  }
}

/// One line's delivered codes: the product label, then each code with its own
/// copy pill.
class _CodeCard extends StatelessWidget {
  const _CodeCard({
    required this.label,
    required this.codes,
    required this.copiedCode,
    required this.onCopy,
    required this.l10n,
  });

  final String label;
  final List<String> codes;
  final String? copiedCode;
  final ValueChanged<String> onCopy;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label,
              style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: colors.textDim)),
          for (final code in codes) ...[
            const SizedBox(height: 8),
            _codeRow(code, copiedCode == code, colors),
          ],
        ],
      ),
    );
  }

  Widget _codeRow(String code, bool copied, AppColors colors) {
    return Row(
      children: [
        Expanded(
          child: Text(
            code,
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w600,
              letterSpacing: 1,
              fontFeatures: const [FontFeature.tabularFigures()],
              color: colors.text,
            ),
          ),
        ),
        const SizedBox(width: 10),
        GestureDetector(
          onTap: () => onCopy(code),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 7),
            decoration: BoxDecoration(
              color: copied ? const Color(0x293FA37A) : AppTokens.cta,
              borderRadius: BorderRadius.circular(AppTokens.rPill),
            ),
            child: Text(
              copied ? l10n.copiedLabel : l10n.copyLabel,
              style: TextStyle(
                fontSize: 12.5,
                fontWeight: FontWeight.w800,
                color: copied ? const Color(0xFF3FA37A) : Colors.white,
              ),
            ),
          ),
        ),
      ],
    );
  }
}

class _ProcessingNote extends StatelessWidget {
  const _ProcessingNote({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.schedule_rounded,
              size: 20, color: AppTokens.brand1),
          const SizedBox(width: 11),
          Expanded(
            child: Text(
              text,
              style: TextStyle(
                  fontSize: 13, height: 1.5, color: colors.textDim),
            ),
          ),
        ],
      ),
    );
  }
}
