import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../browse/domain/entities/app_notification.dart';
import '../../../browse/presentation/providers.dart';
import '../../../browse/presentation/screens/notifications_screen.dart';
import '../../../wallet/presentation/providers.dart';

/// Kinds that pop a banner at the top of the screen (client feedback #64).
const _popupTypes = {
  AppNotificationType.walletTopUp,
  AppNotificationType.walletTopUpRejected,
};

/// Unread top-up notes newer than [since], oldest first — the ones to pop.
/// [since] starts at app launch and advances to the newest popped note, so a
/// note never pops twice and old ones never pop.
List<AppNotification> newTopUpPopups(
  List<AppNotification> items,
  DateTime since,
) {
  final fresh =
      items
          .where(
            (n) =>
                _popupTypes.contains(n.type) &&
                !n.isRead &&
                n.createdAt.isAfter(since),
          )
          .toList()
        ..sort((a, b) => a.createdAt.compareTo(b.createdAt));
  return fresh;
}

/// The pop-up baseline: the newest notification time in [items], or the epoch
/// when the inbox is empty.
DateTime topUpBaseline(List<AppNotification> items) {
  var max = DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
  for (final n in items) {
    if (n.createdAt.isAfter(max)) max = n.createdAt;
  }
  return max;
}

/// Wraps the app shell: polls the unread-notification count while the app is
/// in the foreground and, when it rises, shows a banner at the top of the
/// screen for a newly approved/rejected wallet top-up and refreshes the wallet.
/// Works without FCM push (which isn't configured for every build).
class TopUpNotificationListener extends ConsumerStatefulWidget {
  const TopUpNotificationListener({super.key, required this.child});

  final Widget child;

  @override
  ConsumerState<TopUpNotificationListener> createState() =>
      _TopUpNotificationListenerState();
}

class _TopUpNotificationListenerState
    extends ConsumerState<TopUpNotificationListener>
    with WidgetsBindingObserver {
  static const _pollInterval = Duration(seconds: 20);
  static const _showFor = Duration(seconds: 6);

  Timer? _timer;
  Timer? _hideTimer;
  int? _lastUnread;

  /// Newest notification time already in the inbox (server clock), seeded on
  /// the first check; null until then. Not the device clock, which may run
  /// ahead of the server and hide notes created right after launch.
  DateTime? _since;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _check();
    _start();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _timer?.cancel();
    _hideTimer?.cancel();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _check();
      _start();
    } else {
      _timer?.cancel();
      _timer = null;
    }
  }

  void _start() => _timer ??= Timer.periodic(_pollInterval, (_) => _check());

  Future<void> _check() async {
    final repo = ref.read(notificationRepositoryProvider);
    final count = (await repo.unreadCount()).match((_) => null, (c) => c);
    if (!mounted || count == null) return;
    final first = _since == null;
    final rose = _lastUnread != null && count > _lastUnread!;
    _lastUnread = count;
    if (!first && !rose) return;
    final listed = await repo.listNotifications();
    if (!mounted) return;
    final items = listed.match((_) => null, (l) => l);
    if (items == null) return; // retry the baseline on the next tick
    if (first) {
      // First look at the inbox: everything in it is "old" — no banner.
      _since = topUpBaseline(items);
      return;
    }
    ref.invalidate(unreadNotificationsCountProvider);
    final fresh = newTopUpPopups(items, _since!);
    if (fresh.isEmpty) return;
    _since = fresh.last.createdAt;
    ref.invalidate(walletProvider);
    ref.invalidate(topUpRequestsProvider);
    ref.invalidate(notificationsProvider);
    _show(fresh.last);
  }

  void _show(AppNotification n) {
    final messenger = ScaffoldMessenger.maybeOf(context);
    if (messenger == null) return;
    final l10n = AppLocalizations.of(context);
    final bad = n.type == AppNotificationType.walletTopUpRejected;
    messenger
      ..hideCurrentMaterialBanner()
      ..showMaterialBanner(
        MaterialBanner(
          leading: Icon(
            bad ? Icons.cancel_rounded : Icons.account_balance_wallet_rounded,
            color: bad ? Colors.redAccent : Colors.green,
          ),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                notificationTitle(n, l10n),
                style: const TextStyle(fontWeight: FontWeight.w700),
              ),
              Text(notificationBody(n, l10n)),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () {
                messenger.hideCurrentMaterialBanner();
                context.push('/wallet');
              },
              child: Text(l10n.walletTitle),
            ),
          ],
        ),
      );
    _hideTimer?.cancel();
    _hideTimer = Timer(_showFor, () {
      if (mounted) messenger.hideCurrentMaterialBanner();
    });
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
