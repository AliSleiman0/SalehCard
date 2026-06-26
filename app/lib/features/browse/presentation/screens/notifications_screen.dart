import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../domain/entities/app_notification.dart';
import '../providers.dart';

/// In-app notifications list. Watches [notificationsProvider] (a stub today);
/// each row's title/body is localized by [AppNotification.type] so EN/AR both
/// render. Shows a designed empty state when the list is empty.
class NotificationsScreen extends ConsumerWidget {
  const NotificationsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final notificationsAsync = ref.watch(notificationsProvider);

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.notificationsTitle),
      ),
      body: notificationsAsync.when(
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
                  onPressed: () => ref.invalidate(notificationsProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (items) {
          if (items.isEmpty) {
            return EmptyState(
              icon: Icons.notifications_none_rounded,
              title: l10n.notificationsEmptyTitle,
              message: l10n.notificationsEmptySub,
            );
          }
          return ListView.separated(
            padding: const EdgeInsets.symmetric(vertical: 8),
            itemCount: items.length,
            separatorBuilder: (_, _) =>
                Divider(height: 1, color: colors.border),
            itemBuilder: (_, i) => _NotificationRow(item: items[i], l10n: l10n),
          );
        },
      ),
    );
  }
}

class _NotificationRow extends StatelessWidget {
  const _NotificationRow({required this.item, required this.l10n});

  final AppNotification item;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              gradient: AppTokens.brandGradient,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(_iconFor(item.type), color: Colors.white, size: 20),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        _titleFor(item, l10n),
                        style: TextStyle(
                          fontSize: 14.5,
                          fontWeight: FontWeight.w800,
                          color: colors.text,
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Text(
                      _shortDate(item.createdAt),
                      style: TextStyle(fontSize: 12, color: colors.textFaint),
                    ),
                  ],
                ),
                const SizedBox(height: 3),
                Text(
                  _bodyFor(item, l10n),
                  style: TextStyle(
                      fontSize: 13, height: 1.35, color: colors.textDim),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  static IconData _iconFor(AppNotificationType type) {
    switch (type) {
      case AppNotificationType.orderCompleted:
        return Icons.check_circle_outline_rounded;
      case AppNotificationType.walletTopUp:
        return Icons.account_balance_wallet_outlined;
      case AppNotificationType.promo:
        return Icons.bolt_rounded;
      case AppNotificationType.general:
        return Icons.notifications_none_rounded;
    }
  }

  static String _titleFor(AppNotification item, AppLocalizations l10n) {
    switch (item.type) {
      case AppNotificationType.orderCompleted:
        return l10n.notifOrderCompletedTitle;
      case AppNotificationType.walletTopUp:
        return l10n.notifWalletTopUpTitle;
      case AppNotificationType.promo:
        return l10n.notifPromoTitle;
      case AppNotificationType.general:
        return item.title;
    }
  }

  static String _bodyFor(AppNotification item, AppLocalizations l10n) {
    switch (item.type) {
      case AppNotificationType.orderCompleted:
        return l10n.notifOrderCompletedBody;
      case AppNotificationType.walletTopUp:
        return l10n.notifWalletTopUpBody;
      case AppNotificationType.promo:
        return l10n.notifPromoBody;
      case AppNotificationType.general:
        return item.body;
    }
  }

  static String _shortDate(DateTime at) {
    final local = at.toLocal();
    final m = local.month.toString().padLeft(2, '0');
    final d = local.day.toString().padLeft(2, '0');
    return '${local.year}-$m-$d';
  }
}
