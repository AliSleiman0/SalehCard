import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../domain/entities/order.dart';
import '../providers.dart';

/// Order success / processing screen. Uses the [order] passed via router `extra`
/// when available (avoids a refetch right after placing it); otherwise loads it
/// by [id]. Completed code orders show the delivered code with a copy button;
/// account_credit / transfer orders show the processing variant.
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
        loading: () => const Center(child: CircularProgressIndicator()),
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
  bool _copied = false;

  Future<void> _copy(String code) async {
    await Clipboard.setData(ClipboardData(text: code));
    if (!mounted) return;
    setState(() => _copied = true);
    Future.delayed(const Duration(milliseconds: 1600), () {
      if (mounted) setState(() => _copied = false);
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final order = widget.order;
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final processing = order.isProcessing || !order.hasDeliveredCode;

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
        if (!processing && order.hasDeliveredCode)
          _CodeCard(
            label: _codeLabel(order, localeCode, l10n),
            code: order.fulfillment.deliveredCode!,
            copied: _copied,
            onCopy: () => _copy(order.fulfillment.deliveredCode!),
            l10n: l10n,
          )
        else
          _ProcessingNote(l10n: l10n),
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

  String _codeLabel(Order order, String localeCode, AppLocalizations l10n) {
    for (final item in order.items) {
      if (item.fulfillmentType == 'code') {
        final title = item.title.resolve(localeCode);
        final denom = item.denomination;
        if (title.isEmpty) return l10n.giftCardLabel;
        return denom.isEmpty ? title : '$title · $denom';
      }
    }
    return l10n.giftCardLabel;
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

class _CodeCard extends StatelessWidget {
  const _CodeCard({
    required this.label,
    required this.code,
    required this.copied,
    required this.onCopy,
    required this.l10n,
  });

  final String label;
  final String code;
  final bool copied;
  final VoidCallback onCopy;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.deliveredCodesLabel,
            style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w800,
                color: colors.text)),
        const SizedBox(height: 11),
        Container(
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
              const SizedBox(height: 8),
              Row(
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
                    onTap: onCopy,
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 14, vertical: 7),
                      decoration: BoxDecoration(
                        color: copied
                            ? const Color(0x293FA37A)
                            : AppTokens.cta,
                        borderRadius: BorderRadius.circular(AppTokens.rPill),
                      ),
                      child: Text(
                        copied ? l10n.copiedLabel : l10n.copyLabel,
                        style: TextStyle(
                          fontSize: 12.5,
                          fontWeight: FontWeight.w800,
                          color:
                              copied ? const Color(0xFF3FA37A) : Colors.white,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _ProcessingNote extends StatelessWidget {
  const _ProcessingNote({required this.l10n});

  final AppLocalizations l10n;

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
              l10n.orderProcessingNote,
              style: TextStyle(
                  fontSize: 13, height: 1.5, color: colors.textDim),
            ),
          ),
        ],
      ),
    );
  }
}
