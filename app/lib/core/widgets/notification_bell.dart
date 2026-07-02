import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/browse/presentation/providers.dart';

/// The notification bell: pushes `/notifications` and overlays an amber unread
/// count fed by [unreadNotificationsCountProvider]. The badge hides at zero and
/// on load/error — it must never surface an error state.
class NotificationBell extends ConsumerWidget {
  const NotificationBell({super.key, this.tooltip, this.color});

  final String? tooltip;
  final Color? color;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final count = ref
        .watch(unreadNotificationsCountProvider)
        .maybeWhen(data: (value) => value, orElse: () => 0);

    return Stack(
      clipBehavior: Clip.none,
      children: [
        IconButton(
          tooltip: tooltip,
          icon: Icon(Icons.notifications_none_rounded, color: color),
          onPressed: () => context.push('/notifications'),
        ),
        if (count > 0)
          Positioned(
            top: 4,
            right: 4,
            child: IgnorePointer(
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 2),
                constraints: const BoxConstraints(minWidth: 18),
                decoration: BoxDecoration(
                  color: Colors.amber,
                  borderRadius: BorderRadius.circular(9),
                ),
                child: Text(
                  count > 9 ? '9+' : '$count',
                  textAlign: TextAlign.center,
                  style: const TextStyle(
                    fontSize: 10.5,
                    fontWeight: FontWeight.w800,
                    color: Colors.black87,
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}
