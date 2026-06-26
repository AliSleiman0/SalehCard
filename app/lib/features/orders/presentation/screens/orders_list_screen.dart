import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../../core/widgets/status_badge.dart';
import '../../../checkout/domain/entities/order.dart';
import '../../../checkout/presentation/providers.dart';

/// A client-side status filter for the order history list.
enum _OrderFilter { all, completed, processing, refunded }

/// Order history. Watches [ordersProvider] (newest-first) with All / Completed /
/// Processing / Refunded filter tabs. Each row pushes `/orders/{id}`.
class OrdersListScreen extends ConsumerStatefulWidget {
  const OrdersListScreen({super.key});

  @override
  ConsumerState<OrdersListScreen> createState() => _OrdersListScreenState();
}

class _OrdersListScreenState extends ConsumerState<OrdersListScreen> {
  _OrderFilter _filter = _OrderFilter.all;

  bool _matches(Order order) {
    switch (_filter) {
      case _OrderFilter.all:
        return true;
      case _OrderFilter.completed:
        return order.status == OrderStatus.completed;
      case _OrderFilter.processing:
        return order.status == OrderStatus.processing;
      case _OrderFilter.refunded:
        return order.status == OrderStatus.refunded;
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final ordersAsync = ref.watch(ordersProvider);
    final localeCode = ref.watch(localeControllerProvider).languageCode;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.ordersTitle),
      ),
      body: ordersAsync.when(
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
                  onPressed: () => ref.invalidate(ordersProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (orders) {
          final filtered = orders.where(_matches).toList();
          return Column(
            children: [
              _FilterTabs(
                selected: _filter,
                onSelected: (f) => setState(() => _filter = f),
                l10n: l10n,
              ),
              Expanded(
                child: filtered.isEmpty
                    ? EmptyState(
                        icon: Icons.receipt_long_rounded,
                        title: l10n.ordersEmptyTitle,
                        message: l10n.ordersEmptySub,
                        actionLabel: l10n.browseCatalog,
                        onAction: () => context.go('/browse'),
                      )
                    : ListView.separated(
                        padding: const EdgeInsets.fromLTRB(20, 4, 20, 28),
                        itemCount: filtered.length,
                        separatorBuilder: (_, _) => const SizedBox(height: 12),
                        itemBuilder: (context, i) => _OrderRow(
                          order: filtered[i],
                          localeCode: localeCode,
                          l10n: l10n,
                        ),
                      ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _FilterTabs extends StatelessWidget {
  const _FilterTabs({
    required this.selected,
    required this.onSelected,
    required this.l10n,
  });

  final _OrderFilter selected;
  final ValueChanged<_OrderFilter> onSelected;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final tabs = <(_OrderFilter, String)>[
      (_OrderFilter.all, l10n.filterAll),
      (_OrderFilter.completed, l10n.statusCompleted),
      (_OrderFilter.processing, l10n.statusProcessing),
      (_OrderFilter.refunded, l10n.statusRefunded),
    ];
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.fromLTRB(20, 12, 20, 14),
      child: Row(
        children: [
          for (final (filter, label) in tabs)
            Padding(
              padding: const EdgeInsetsDirectional.only(end: 8),
              child: _FilterChip(
                label: label,
                active: filter == selected,
                onTap: () => onSelected(filter),
              ),
            ),
        ],
      ),
    );
  }
}

class _FilterChip extends StatelessWidget {
  const _FilterChip({
    required this.label,
    required this.active,
    required this.onTap,
  });

  final String label;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        decoration: BoxDecoration(
          gradient: active ? AppTokens.brandGradient : null,
          color: active ? null : colors.surface,
          border: active ? null : Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rPill),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w700,
            color: active ? Colors.white : colors.textDim,
          ),
        ),
      ),
    );
  }
}

class _OrderRow extends StatelessWidget {
  const _OrderRow({
    required this.order,
    required this.localeCode,
    required this.l10n,
  });

  final Order order;
  final String localeCode;
  final AppLocalizations l10n;

  (String, Color) _status() {
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

  String _shortDate(DateTime? at) {
    if (at == null) return '';
    final local = at.toLocal();
    final m = local.month.toString().padLeft(2, '0');
    final d = local.day.toString().padLeft(2, '0');
    return '${local.year}-$m-$d';
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final (statusLabel, statusColor) = _status();
    final hasItems = order.items.isNotEmpty;
    final firstTitle =
        hasItems ? order.items.first.title.resolve(localeCode) : '';
    final extraCount = order.items.length - 1;
    final date = _shortDate(order.createdAt);

    return GestureDetector(
      onTap: () => context.push('/orders/${order.id}'),
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: colors.surface,
          border: Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Row(
          children: [
            Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(
                color: hasItems
                    ? ProductChip.tintFor(
                        order.items.first.productId.hashCode.abs())
                    : StatusBadge.neutral,
                borderRadius: BorderRadius.circular(12),
              ),
              alignment: Alignment.center,
              child: Text(
                hasItems ? ProductChip.initialsFor(firstTitle) : '?',
                style: const TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w800,
                    fontSize: 15),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    extraCount > 0 ? '$firstTitle  +$extraCount' : firstTitle,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                        fontSize: 14.5,
                        fontWeight: FontWeight.w800,
                        color: colors.text),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    date.isEmpty
                        ? '#${order.reference}'
                        : '#${order.reference} · $date',
                    style: TextStyle(fontSize: 12.5, color: colors.textDim),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 10),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                StatusBadge(label: statusLabel, color: statusColor),
                const SizedBox(height: 8),
                Text(
                  formatUsd(order.total),
                  style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w800,
                      color: colors.text),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
