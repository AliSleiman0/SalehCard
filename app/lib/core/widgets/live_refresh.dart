import 'dart:async';

import 'package:flutter/widgets.dart';

/// How often a screen the customer is looking at re-checks data that can change
/// on the server without them doing anything (an admin approves a top-up,
/// completes or refunds an order, adjusts the balance). Matches the web (30s).
const liveRefreshInterval = Duration(seconds: 30);

/// Calls [onRefresh] every [interval] while this widget is mounted and the app
/// is in the foreground; pauses in the background and refreshes once on resume.
/// The callback should only invalidate providers — `.when` keeps the current
/// data on screen during the refetch, so nothing flashes.
class LiveRefresh extends StatefulWidget {
  const LiveRefresh({
    super.key,
    required this.onRefresh,
    required this.child,
    this.interval = liveRefreshInterval,
  });

  final VoidCallback onRefresh;
  final Widget child;
  final Duration interval;

  @override
  State<LiveRefresh> createState() => _LiveRefreshState();
}

class _LiveRefreshState extends State<LiveRefresh> {
  Timer? _timer;
  late final AppLifecycleListener _lifecycle;

  @override
  void initState() {
    super.initState();
    _lifecycle = AppLifecycleListener(onStateChange: _onState);
    _start();
  }

  void _onState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      widget.onRefresh();
      _start();
    } else {
      _stop();
    }
  }

  void _start() {
    _timer ??= Timer.periodic(widget.interval, (_) => widget.onRefresh());
  }

  void _stop() {
    _timer?.cancel();
    _timer = null;
  }

  @override
  void dispose() {
    _stop();
    _lifecycle.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
