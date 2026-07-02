import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/ledger_row.dart';
import '../../domain/entities/wallet.dart';
import '../providers.dart';

/// Wallet landing: a gradient balance card, top-up / send-money actions, and the
/// newest-first transaction ledger. Watches [walletProvider] (`GET /wallet`).
///
/// Supports pull-to-refresh and a lightweight 45s poll while the screen is
/// foregrounded, so an admin-approved top-up (or a completed order's refund)
/// appears without navigating away. Polling is paused while the app is
/// backgrounded and fires an immediate refresh on resume.
class WalletScreen extends ConsumerStatefulWidget {
  const WalletScreen({super.key});

  @override
  ConsumerState<WalletScreen> createState() => _WalletScreenState();
}

class _WalletScreenState extends ConsumerState<WalletScreen>
    with WidgetsBindingObserver {
  static const _pollInterval = Duration(seconds: 45);
  Timer? _pollTimer;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _startPolling();
  }

  @override
  void dispose() {
    _stopPolling();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _refresh();
      _startPolling();
    } else {
      _stopPolling();
    }
  }

  void _startPolling() {
    _pollTimer ??= Timer.periodic(_pollInterval, (_) => _refresh());
  }

  void _stopPolling() {
    _pollTimer?.cancel();
    _pollTimer = null;
  }

  /// Re-fetch the balance/ledger and the top-up request history together — an
  /// approval moves both. `.when` keeps the current data on screen during the
  /// refetch (skipLoadingOnRefresh defaults to true), so this does not flash the
  /// full-screen spinner.
  void _refresh() {
    ref.invalidate(walletProvider);
    ref.invalidate(topUpRequestsProvider);
  }

  /// Pull-to-refresh handler: invalidate, then await the wallet refetch so the
  /// pull spinner stays until fresh data is in.
  Future<void> _pullRefresh() async {
    _refresh();
    await ref.read(walletProvider.future);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final walletAsync = ref.watch(walletProvider);
    final requestsAsync = ref.watch(topUpRequestsProvider);

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.walletTitle),
      ),
      body: walletAsync.when(
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
                  onPressed: () => ref.invalidate(walletProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (wallet) => RefreshIndicator(
          onRefresh: _pullRefresh,
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(20, 16, 20, 28),
            children: [
              _BalanceCard(balance: wallet.balance, l10n: l10n),
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: _ActionButton(
                      icon: Icons.add_rounded,
                      label: l10n.topUpCta,
                      onTap: () => context.push('/wallet/topup'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: _ActionButton(
                      icon: Icons.send_rounded,
                      label: l10n.sendMoney,
                      onTap: () => context.push('/wallet/send'),
                    ),
                  ),
                ],
              ),
              // Pending top-up requests: explain why the balance hasn't moved.
              // Renders nothing until pending data is available, so it never
              // blocks the wallet.
              requestsAsync.maybeWhen(
                data: (requests) {
                  final pending = requests
                      .where((r) => r.status == TopUpStatus.pending)
                      .toList();
                  if (pending.isEmpty) return const SizedBox.shrink();
                  final total = pending.fold<double>(
                    0,
                    (sum, r) => sum + r.amount,
                  );
                  return Padding(
                    padding: const EdgeInsets.only(top: 16),
                    child: _PendingTopUpBanner(
                      count: pending.length,
                      total: total,
                      l10n: l10n,
                    ),
                  );
                },
                orElse: () => const SizedBox.shrink(),
              ),
              const SizedBox(height: 24),
              Text(
                l10n.txHistory,
                style: TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w800,
                  color: colors.text,
                ),
              ),
              const SizedBox(height: 4),
              if (wallet.transactions.isEmpty)
                Padding(
                  padding: const EdgeInsets.only(top: 24),
                  child: EmptyState(
                    icon: Icons.account_balance_wallet_outlined,
                    title: l10n.walletEmptyTitle,
                    message: l10n.walletEmptySub,
                    actionLabel: l10n.topUpCta,
                    onAction: () => context.push('/wallet/topup'),
                  ),
                )
              else
                for (final tx in wallet.transactions)
                  LedgerRow(
                    icon: _iconFor(tx.type),
                    title: _labelFor(tx.type, l10n),
                    subtitle: _shortDate(tx.createdAt),
                    amount: tx.amount,
                  ),
            ],
          ),
        ),
      ),
    );
  }

  static IconData _iconFor(WalletTxType type) {
    switch (type) {
      case WalletTxType.topup:
        return Icons.add_rounded;
      case WalletTxType.purchase:
        return Icons.shopping_bag_outlined;
      case WalletTxType.refund:
        return Icons.assignment_return_rounded;
      case WalletTxType.adjustment:
      case WalletTxType.unknown:
        return Icons.tune_rounded;
    }
  }

  static String _labelFor(WalletTxType type, AppLocalizations l10n) {
    switch (type) {
      case WalletTxType.topup:
        return l10n.txTopUp;
      case WalletTxType.purchase:
        return l10n.txPurchase;
      case WalletTxType.refund:
        return l10n.txRefund;
      case WalletTxType.adjustment:
      case WalletTxType.unknown:
        return l10n.txAdjustment;
    }
  }

  static String _shortDate(DateTime? at) {
    if (at == null) return '';
    final local = at.toLocal();
    final m = local.month.toString().padLeft(2, '0');
    final d = local.day.toString().padLeft(2, '0');
    return '${local.year}-$m-$d';
  }
}

/// Compact summary banner shown when the user has pending top-up requests, so an
/// unchanged balance is explained (funding is admin-approved). Taps through to
/// the top-up screen for the full request list + any rejection reasons.
class _PendingTopUpBanner extends StatelessWidget {
  const _PendingTopUpBanner({
    required this.count,
    required this.total,
    required this.l10n,
  });

  final int count;
  final double total;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: () => context.push('/wallet/topup'),
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
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                color: AppTokens.brand1.withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(12),
              ),
              child: const Icon(
                Icons.schedule_rounded,
                color: AppTokens.brand1,
                size: 20,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    l10n.walletPendingTitle(count),
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w800,
                      color: colors.text,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    l10n.walletPendingHint,
                    style: TextStyle(
                      fontSize: 12.5,
                      height: 1.35,
                      color: colors.textDim,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 10),
            Text(
              formatUsd(total),
              style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w800,
                color: AppTokens.brand1,
              ),
            ),
            Icon(Icons.chevron_right_rounded, color: colors.textFaint),
          ],
        ),
      ),
    );
  }
}

class _BalanceCard extends StatelessWidget {
  const _BalanceCard({required this.balance, required this.l10n});

  final double balance;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(22),
      decoration: BoxDecoration(
        gradient: AppTokens.brandGradient,
        borderRadius: BorderRadius.circular(AppTokens.rLg),
        boxShadow: [
          BoxShadow(
            color: AppTokens.brandMid.withValues(alpha: 0.4),
            blurRadius: 32,
            offset: const Offset(0, 14),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.currentBalance,
            style: const TextStyle(
              color: Colors.white70,
              fontSize: 13,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            formatUsd(balance),
            style: const TextStyle(
              color: Colors.white,
              fontSize: 34,
              fontWeight: FontWeight.w800,
            ),
          ),
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: Colors.white24,
              borderRadius: BorderRadius.circular(AppTokens.rPill),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.bolt_rounded, color: Colors.white, size: 16),
                const SizedBox(width: 6),
                Text(
                  l10n.promoTitle,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 12,
                    fontWeight: FontWeight.w700,
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

class _ActionButton extends StatelessWidget {
  const _ActionButton({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 14),
        decoration: BoxDecoration(
          color: colors.surface,
          border: Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Column(
          children: [
            Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                gradient: AppTokens.brandGradient,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(icon, color: Colors.white, size: 20),
            ),
            const SizedBox(height: 8),
            Text(
              label,
              style: TextStyle(
                fontSize: 13.5,
                fontWeight: FontWeight.w800,
                color: colors.text,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
