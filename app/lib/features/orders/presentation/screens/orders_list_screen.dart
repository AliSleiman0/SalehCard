import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_back_button.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/live_refresh.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../../core/widgets/search_field.dart';
import '../../../../core/widgets/status_badge.dart';
import '../../../../core/format/date_time.dart';
import '../../../checkout/domain/entities/order.dart';
import '../../../checkout/presentation/providers.dart';
import '../../domain/order_search.dart';
import '../widgets/balance_line.dart';

/// A client-side status filter for the order history list.
enum _OrderFilter { all, completed, processing, refunded }

/// Order history. Watches [ordersProvider] (newest-first) with a search field
/// (order number, player ID, field values, recipient, product title; see
/// [orderMatchesQuery]) and All / Completed / Processing / Refunded filter
/// tabs. The API returns the whole history, so both filter on the client.
/// Each row pushes `/orders/{id}`.
class OrdersListScreen extends ConsumerStatefulWidget {
  const OrdersListScreen({super.key});

  @override
  ConsumerState<OrdersListScreen> createState() => _OrdersListScreenState();
}

class _OrdersListScreenState extends ConsumerState<OrdersListScreen> {
  _OrderFilter _filter = _OrderFilter.all;
  final _search = TextEditingController();
  String _query = '';

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  void _clearSearch() {
    _search.clear();
    setState(() => _query = '');
  }

  /// Pull-to-refresh: invalidate then await the refetch so the pull spinner
  /// stays until fresh order history is in.
  Future<void> _refresh() async {
    ref.invalidate(ordersProvider);
    await ref.read(ordersProvider.future);
  }

  bool _matches(Order order) =>
      _matchesStatus(order) && orderMatchesQuery(order, _query);

  bool _matchesStatus(Order order) {
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

    return LiveRefresh(
      onRefresh: () => ref.invalidate(ordersProvider),
      child: Scaffold(
        backgroundColor: colors.bg,
        appBar: AppBar(
          backgroundColor: colors.topbar,
          // Normally pushed from the Menu tab, but order detail's fallback reaches
          // it via go (order-success → "View order" → back) as the stack root,
          // which would leave no back arrow at all.
          leading: const AppBackButton(fallback: '/account'),
          title: Text(l10n.ordersTitle),
        ),
        body: ordersAsync.when(
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
                    onPressed: () => ref.invalidate(ordersProvider),
                    child: Text(l10n.retry),
                  ),
                ],
              ),
            ),
          ),
          data: (orders) {
            final filtered = orders.where(_matches).toList();
            final searching = normalizeOrderQuery(_query).isNotEmpty;
            return Column(
              children: [
                if (orders.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(20, 12, 20, 0),
                    child: _SearchBox(
                      controller: _search,
                      hint: l10n.ordersSearchHint,
                      clearLabel: l10n.ordersSearchClear,
                      onChanged: (v) => setState(() => _query = v),
                      onClear: _clearSearch,
                    ),
                  ),
                _FilterTabs(
                  selected: _filter,
                  onSelected: (f) => setState(() => _filter = f),
                  l10n: l10n,
                ),
                Expanded(
                  child: RefreshIndicator(
                    onRefresh: _refresh,
                    // Keep the empty state pullable: a user with no matching
                    // orders should still be able to pull to check for new ones.
                    child: filtered.isEmpty
                        ? LayoutBuilder(
                            builder: (context, constraints) =>
                                SingleChildScrollView(
                                  physics:
                                      const AlwaysScrollableScrollPhysics(),
                                  child: ConstrainedBox(
                                    constraints: BoxConstraints(
                                      minHeight: constraints.maxHeight,
                                    ),
                                    child: searching
                                        ? EmptyState(
                                            icon: Icons.search_off_rounded,
                                            title: l10n.ordersSearchNoneTitle(
                                              _query.trim(),
                                            ),
                                            message: l10n.ordersSearchNoneSub,
                                            actionLabel: l10n.ordersSearchClear,
                                            onAction: _clearSearch,
                                          )
                                        : EmptyState(
                                            icon: Icons.receipt_long_rounded,
                                            title: l10n.ordersEmptyTitle,
                                            message: l10n.ordersEmptySub,
                                            actionLabel: l10n.browseCatalog,
                                            onAction: () =>
                                                context.go('/browse'),
                                          ),
                                  ),
                                ),
                          )
                        : ListView.separated(
                            physics: const AlwaysScrollableScrollPhysics(),
                            padding: const EdgeInsets.fromLTRB(20, 4, 20, 28),
                            itemCount: filtered.length,
                            separatorBuilder: (_, _) =>
                                const SizedBox(height: 12),
                            itemBuilder: (context, i) => _OrderRow(
                              order: filtered[i],
                              localeCode: localeCode,
                              l10n: l10n,
                            ),
                          ),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}

/// The rounded search pill above the filter tabs (search icon + the shared
/// [SearchField], which brings its own clear button).
class _SearchBox extends StatelessWidget {
  const _SearchBox({
    required this.controller,
    required this.hint,
    required this.clearLabel,
    required this.onChanged,
    required this.onClear,
  });

  final TextEditingController controller;
  final String hint;
  final String clearLabel;
  final ValueChanged<String> onChanged;
  final VoidCallback onClear;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      height: 48,
      padding: const EdgeInsetsDirectional.only(start: 14, end: 4),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rPill),
      ),
      child: Row(
        children: [
          Icon(Icons.search_rounded, size: 20, color: colors.textFaint),
          const SizedBox(width: 8),
          Expanded(
            child: SearchField(
              controller: controller,
              hint: hint,
              clearTooltip: clearLabel,
              onChanged: onChanged,
              onClear: onClear,
            ),
          ),
        ],
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

  String _shortDate(DateTime? at) => formatDateTime(at);

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final (statusLabel, statusColor) = _status();
    final hasItems = order.items.isNotEmpty;
    final firstTitle = hasItems
        ? order.items.first.title.resolve(localeCode)
        : '';
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
                        order.items.first.productId.hashCode.abs(),
                      )
                    : StatusBadge.neutral,
                borderRadius: BorderRadius.circular(12),
              ),
              alignment: Alignment.center,
              child: Text(
                hasItems ? ProductChip.initialsFor(firstTitle) : '?',
                style: const TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.w800,
                  fontSize: 15,
                ),
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
                      color: colors.text,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    date.isEmpty
                        ? '#${order.reference}'
                        : '#${order.reference} · $date',
                    style: TextStyle(fontSize: 12.5, color: colors.textDim),
                  ),
                  if (order.walletBalanceAfter != null) ...[
                    const SizedBox(height: 2),
                    BalanceLine(order: order, fontSize: 11.5),
                  ],
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
                    color: colors.text,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
