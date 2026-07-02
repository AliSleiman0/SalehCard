import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/money_row.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../../core/widgets/status_badge.dart';
import '../../../checkout/domain/entities/order.dart';
import '../../../checkout/presentation/providers.dart';

/// Order detail, wired to `GET /orders/{id}` via [orderDetailProvider]. Renders
/// the delivered-code, processing, and refunded (display-only) variants plus the
/// status timeline.
class OrderDetailScreen extends ConsumerWidget {
  const OrderDetailScreen({super.key, required this.id});

  final String id;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final orderAsync = ref.watch(orderDetailProvider(id));
    final localeCode = ref.watch(localeControllerProvider).languageCode;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: orderAsync.maybeWhen(
          data: (o) => Text('${l10n.orderLabel} #${o.reference}'),
          orElse: () => Text(l10n.orderLabel),
        ),
      ),
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
                  onPressed: () => ref.invalidate(orderDetailProvider(id)),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (order) => RefreshIndicator(
          // The user watches this screen while an order sits in `processing`
          // waiting for a code drop / manual completion — let them pull to
          // check without leaving.
          onRefresh: () async {
            ref.invalidate(orderDetailProvider(id));
            await ref.read(orderDetailProvider(id).future);
          },
          child: _Body(order: order, localeCode: localeCode, l10n: l10n),
        ),
      ),
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({
    required this.order,
    required this.localeCode,
    required this.l10n,
  });

  final Order order;
  final String localeCode;
  final AppLocalizations l10n;

  (String, Color) _status(BuildContext context) {
    switch (order.status) {
      case OrderStatus.completed:
        return (l10n.statusCompleted, StatusBadge.success);
      case OrderStatus.processing:
        return (l10n.statusProcessing, StatusBadge.warning);
      case OrderStatus.pending:
        return (l10n.statusPending, StatusBadge.neutral);
      case OrderStatus.failed:
        return (l10n.statusFailed, StatusBadge.danger);
      case OrderStatus.refunded:
        return (l10n.statusRefunded, StatusBadge.danger);
      case OrderStatus.unknown:
        return (order.status.name, StatusBadge.neutral);
    }
  }

  /// Localise a raw timeline status string where it maps to a known status,
  /// else return the raw value unchanged.
  String _statusLabel(String raw) {
    switch (orderStatusFromString(raw)) {
      case OrderStatus.completed:
        return l10n.statusCompleted;
      case OrderStatus.processing:
        return l10n.statusProcessing;
      case OrderStatus.pending:
        return l10n.statusPending;
      case OrderStatus.failed:
        return l10n.statusFailed;
      case OrderStatus.refunded:
        return l10n.statusRefunded;
      case OrderStatus.unknown:
        return raw;
    }
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final (label, color) = _status(context);

    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 28),
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            StatusBadge(label: label, color: color),
            Text(
              order.paymentMethod.toUpperCase(),
              style: TextStyle(
                fontSize: 12.5,
                fontWeight: FontWeight.w700,
                color: colors.textFaint,
              ),
            ),
          ],
        ),
        const SizedBox(height: 20),
        if (order.hasDeliveredCode) ...[
          _CodeCard(code: order.fulfillment.deliveredCode!, l10n: l10n),
          const SizedBox(height: 20),
        ] else if (order.isProcessing) ...[
          _ProcessingNote(l10n: l10n),
          const SizedBox(height: 20),
        ] else if (order.status == OrderStatus.refunded) ...[
          _RefundedNote(amount: order.total, l10n: l10n),
          const SizedBox(height: 20),
        ],
        Text(
          l10n.orderItemsLabel,
          style: TextStyle(
            fontSize: 15,
            fontWeight: FontWeight.w800,
            color: colors.text,
          ),
        ),
        const SizedBox(height: 10),
        Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: colors.surface,
            border: Border.all(color: colors.border),
            borderRadius: BorderRadius.circular(AppTokens.rMd),
          ),
          child: Column(
            children: [
              for (final item in order.items)
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: Row(
                    children: [
                      Container(
                        width: 38,
                        height: 38,
                        decoration: BoxDecoration(
                          color: ProductChip.tintFor(
                            item.productId.hashCode.abs(),
                          ),
                          borderRadius: BorderRadius.circular(11),
                        ),
                        alignment: Alignment.center,
                        child: Text(
                          ProductChip.initialsFor(
                            item.title.resolve(localeCode),
                          ),
                          style: const TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.w800,
                            fontSize: 13,
                          ),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              '${item.title.resolve(localeCode)}  ×${item.qty}',
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 14,
                                fontWeight: FontWeight.w700,
                                color: colors.text,
                              ),
                            ),
                            if (item.denomination.isNotEmpty)
                              Text(
                                item.denomination,
                                style: TextStyle(
                                  fontSize: 12.5,
                                  color: colors.textDim,
                                ),
                              ),
                          ],
                        ),
                      ),
                      Text(
                        formatUsd(item.lineTotal),
                        style: TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w800,
                          color: colors.text,
                        ),
                      ),
                    ],
                  ),
                ),
              Divider(height: 1, color: colors.border),
              const SizedBox(height: 8),
              MoneyRow(label: l10n.subtotalLabel, value: order.subtotal),
              MoneyRow(
                label: l10n.totalLabel,
                value: order.total,
                emphasized: true,
              ),
            ],
          ),
        ),
        if (order.fulfillment.statusTimeline.isNotEmpty) ...[
          const SizedBox(height: 24),
          Text(
            l10n.orderTimelineLabel,
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w800,
              color: colors.text,
            ),
          ),
          const SizedBox(height: 10),
          _Timeline(
            events: order.fulfillment.statusTimeline,
            labelOf: _statusLabel,
          ),
        ],
      ],
    );
  }
}

class _RefundedNote extends StatelessWidget {
  const _RefundedNote({required this.amount, required this.l10n});

  final double amount;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: StatusBadge.danger.withValues(alpha: 0.08),
        border: Border.all(color: StatusBadge.danger.withValues(alpha: 0.35)),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(
            Icons.assignment_return_rounded,
            size: 20,
            color: StatusBadge.danger,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.orderRefundedNote,
                  style: TextStyle(
                    fontSize: 13,
                    height: 1.4,
                    color: colors.textDim,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  formatUsd(amount),
                  style: const TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w800,
                    color: StatusBadge.danger,
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

/// A vertical status timeline: each event is a dot + connector line, a localised
/// status label, the note, and the time (when present).
class _Timeline extends StatelessWidget {
  const _Timeline({required this.events, required this.labelOf});

  final List<TimelineEvent> events;
  final String Function(String) labelOf;

  String _time(DateTime? at) {
    if (at == null) return '';
    final local = at.toLocal();
    final m = local.month.toString().padLeft(2, '0');
    final d = local.day.toString().padLeft(2, '0');
    final h = local.hour.toString().padLeft(2, '0');
    final min = local.minute.toString().padLeft(2, '0');
    return '${local.year}-$m-$d $h:$min';
  }

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
          for (var i = 0; i < events.length; i++)
            IntrinsicHeight(
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Column(
                    children: [
                      Container(
                        width: 10,
                        height: 10,
                        margin: const EdgeInsets.only(top: 3),
                        decoration: const BoxDecoration(
                          gradient: AppTokens.brandGradient,
                          shape: BoxShape.circle,
                        ),
                      ),
                      if (i != events.length - 1)
                        Expanded(
                          child: Container(width: 2, color: colors.border),
                        ),
                    ],
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Padding(
                      padding: EdgeInsets.only(
                        bottom: i == events.length - 1 ? 0 : 16,
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            labelOf(events[i].status),
                            style: TextStyle(
                              fontSize: 13.5,
                              fontWeight: FontWeight.w800,
                              color: colors.text,
                            ),
                          ),
                          if (events[i].note.isNotEmpty) ...[
                            const SizedBox(height: 2),
                            Text(
                              events[i].note,
                              style: TextStyle(
                                fontSize: 12.5,
                                height: 1.35,
                                color: colors.textDim,
                              ),
                            ),
                          ],
                          if (_time(events[i].at).isNotEmpty) ...[
                            const SizedBox(height: 2),
                            Text(
                              _time(events[i].at),
                              style: TextStyle(
                                fontSize: 11.5,
                                color: colors.textFaint,
                              ),
                            ),
                          ],
                        ],
                      ),
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

class _CodeCard extends StatelessWidget {
  const _CodeCard({required this.code, required this.l10n});

  final String code;
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
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.deliveredCodesLabel,
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w700,
              color: colors.textDim,
            ),
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: Text(
                  code,
                  style: TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                    color: colors.text,
                  ),
                ),
              ),
              GestureDetector(
                onTap: () {
                  Clipboard.setData(ClipboardData(text: code));
                  ScaffoldMessenger.of(
                    context,
                  ).showSnackBar(SnackBar(content: Text(l10n.copiedLabel)));
                },
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 14,
                    vertical: 8,
                  ),
                  decoration: BoxDecoration(
                    gradient: AppTokens.brandGradient,
                    borderRadius: BorderRadius.circular(AppTokens.rPill),
                  ),
                  child: Text(
                    l10n.copyLabel,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 13,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
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
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        children: [
          const Icon(Icons.schedule_rounded, size: 20, color: AppTokens.brand1),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              l10n.orderProcessingNote,
              style: TextStyle(
                fontSize: 13,
                height: 1.4,
                color: colors.textDim,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
