import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../checkout/presentation/providers.dart' show ordersProvider;
import '../../../wallet/presentation/providers.dart'
    show walletProvider, topUpRequestsProvider;
import '../../domain/entities/payment_intent.dart';
import '../providers.dart';

/// Waiting-for-payment screen for an on-chain USDT deposit. Shows the QR + exact
/// amount + TRC20-only address to send to, a live countdown to expiry, and polls
/// `GET /payments/intents/{id}` every few seconds (lifecycle-aware, like the
/// wallet screen) until the intent is confirmed or expired.
///
/// Reached with the freshly-created [PaymentIntent] as the router `extra`, so
/// the first frame renders instantly; subsequent state comes from the poll.
class UsdtDepositScreen extends ConsumerStatefulWidget {
  const UsdtDepositScreen({super.key, required this.intent});

  final PaymentIntent intent;

  @override
  ConsumerState<UsdtDepositScreen> createState() => _UsdtDepositScreenState();
}

class _UsdtDepositScreenState extends ConsumerState<UsdtDepositScreen>
    with WidgetsBindingObserver {
  static const _pollInterval = Duration(seconds: 7);
  Timer? _pollTimer;
  Timer? _tickTimer; // 1s countdown repaint
  bool _navigated = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _startPolling();
    _tickTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (mounted) setState(() {});
    });
  }

  @override
  void dispose() {
    _stopPolling();
    _tickTimer?.cancel();
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

  void _startPolling() =>
      _pollTimer ??= Timer.periodic(_pollInterval, (_) => _refresh());

  void _stopPolling() {
    _pollTimer?.cancel();
    _pollTimer = null;
  }

  void _refresh() => ref.invalidate(paymentIntentProvider(widget.intent.id));

  /// On a confirmed payment, stop polling, refresh what the deposit funded
  /// (wallet or order), and route the user onward.
  void _onConfirmed(PaymentIntent intent) {
    if (_navigated) return;
    _navigated = true;
    _stopPolling();
    ref.invalidate(walletProvider);
    ref.invalidate(topUpRequestsProvider);
    ref.invalidate(ordersProvider);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (intent.purpose == PaymentPurpose.order && intent.orderId != null) {
        context.pushReplacement('/orders/${intent.orderId}');
      } else {
        context.go('/wallet');
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    // Fall back to the seed intent passed via `extra` until the first poll lands.
    final async = ref.watch(paymentIntentProvider(widget.intent.id));
    final intent = async.asData?.value ?? widget.intent;

    if (intent.isConfirmed) {
      _onConfirmed(intent);
      return _TerminalPanel(
        icon: Icons.check_circle_rounded,
        color: AppTokens.accent,
        title: l10n.usdtConfirmedTitle,
        body: l10n.usdtConfirmedBody,
      );
    }
    if (intent.isExpired) {
      return _TerminalPanel(
        icon: Icons.timer_off_rounded,
        color: AppTokens.danger,
        title: l10n.usdtExpiredTitle,
        body: l10n.usdtExpiredBody,
        actionLabel: l10n.commonDone,
        onAction: () => context.pop(),
      );
    }

    final confirming = intent.status == PaymentIntentStatus.confirming;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.usdtDepositTitle),
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(20, 18, 20, 28),
        children: [
          if (confirming) _ConfirmingBanner(l10n: l10n),
          Center(
            child: Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(AppTokens.rMd),
              ),
              child: QrImageView(
                data: intent.address,
                size: 190,
                backgroundColor: Colors.white,
                // Errors here would only be a malformed address (never in
                // practice — it's server-derived), so keep the default view.
              ),
            ),
          ),
          const SizedBox(height: 22),
          Text(
            l10n.usdtSendExactly,
            style: TextStyle(
              fontSize: 13.5,
              fontWeight: FontWeight.w700,
              color: colors.textDim,
            ),
          ),
          const SizedBox(height: 8),
          _CopyField(
            label: l10n.usdtAmountLabel,
            value: formatUsdtAmount(intent.amountUsd),
            display: '${formatUsdtAmount(intent.amountUsd)} USDT',
            copiedMsg: l10n.usdtAmountCopied,
          ),
          const SizedBox(height: 12),
          _CopyField(
            label: l10n.usdtAddressLabel(intent.network.toUpperCase()),
            value: intent.address,
            display: intent.address,
            mono: true,
            copiedMsg: l10n.usdtAddressCopied,
          ),
          const SizedBox(height: 18),
          _NetworkWarning(l10n: l10n, network: intent.network.toUpperCase()),
          const SizedBox(height: 18),
          _ExpiryRow(l10n: l10n, expiresAt: intent.expiresAt),
          const SizedBox(height: 20),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const AppSpinner(size: 16),
              const SizedBox(width: 10),
              Text(
                l10n.usdtWaiting,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: colors.textFaint,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _ConfirmingBanner extends StatelessWidget {
  const _ConfirmingBanner({required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 18),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: AppTokens.accent.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        children: [
          const AppSpinner(size: 16),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              l10n.usdtConfirming,
              style: const TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w700,
                color: AppTokens.accent,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// A labeled value row with a copy-to-clipboard button (order-detail pattern).
class _CopyField extends StatefulWidget {
  const _CopyField({
    required this.label,
    required this.value,
    required this.display,
    required this.copiedMsg,
    this.mono = false,
  });

  final String label;
  final String value;
  final String display;
  final String copiedMsg;
  final bool mono;

  @override
  State<_CopyField> createState() => _CopyFieldState();
}

class _CopyFieldState extends State<_CopyField> {
  bool _copied = false;

  Future<void> _copy() async {
    await Clipboard.setData(ClipboardData(text: widget.value));
    if (!mounted) return;
    setState(() => _copied = true);
    Timer(const Duration(milliseconds: 1600), () {
      if (mounted) setState(() => _copied = false);
    });
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: _copy,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        decoration: BoxDecoration(
          color: colors.surface,
          border: Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    widget.label,
                    style: TextStyle(
                      fontSize: 11.5,
                      fontWeight: FontWeight.w700,
                      color: colors.textFaint,
                    ),
                  ),
                  const SizedBox(height: 3),
                  Text(
                    widget.display,
                    style: TextStyle(
                      fontSize: widget.mono ? 12.5 : 15,
                      fontWeight: FontWeight.w800,
                      color: colors.text,
                      fontFamily: widget.mono ? 'monospace' : null,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 10),
            Icon(
              _copied ? Icons.check_rounded : Icons.copy_rounded,
              size: 18,
              color: _copied ? AppTokens.accent : colors.textDim,
            ),
          ],
        ),
      ),
    );
  }
}

class _NetworkWarning extends StatelessWidget {
  const _NetworkWarning({required this.l10n, required this.network});

  final AppLocalizations l10n;
  final String network;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: AppTokens.danger.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        children: [
          const Icon(Icons.warning_amber_rounded,
              size: 18, color: AppTokens.danger),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              l10n.usdtNetworkWarning(network),
              style: const TextStyle(
                fontSize: 12.5,
                fontWeight: FontWeight.w700,
                color: AppTokens.danger,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ExpiryRow extends StatelessWidget {
  const _ExpiryRow({required this.l10n, required this.expiresAt});

  final AppLocalizations l10n;
  final DateTime? expiresAt;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final remaining = expiresAt?.difference(DateTime.now()) ?? Duration.zero;
    final secs = remaining.inSeconds.clamp(0, 24 * 3600);
    final mm = (secs ~/ 60).toString().padLeft(2, '0');
    final ss = (secs % 60).toString().padLeft(2, '0');
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Icon(Icons.schedule_rounded, size: 16, color: colors.textFaint),
        const SizedBox(width: 8),
        Text(
          l10n.usdtExpiresIn('$mm:$ss'),
          style: TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w700,
            color: colors.textDim,
          ),
        ),
      ],
    );
  }
}

class _TerminalPanel extends StatelessWidget {
  const _TerminalPanel({
    required this.icon,
    required this.color,
    required this.title,
    required this.body,
    this.actionLabel,
    this.onAction,
  });

  final IconData icon;
  final Color color;
  final String title;
  final String body;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Scaffold(
      backgroundColor: colors.bg,
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(28),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 64, color: color),
              const SizedBox(height: 18),
              Text(
                title,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w800,
                  color: colors.text,
                ),
              ),
              const SizedBox(height: 10),
              Text(
                body,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                  color: colors.textDim,
                ),
              ),
              if (actionLabel != null) ...[
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: onAction,
                  style: FilledButton.styleFrom(
                    minimumSize: const Size(180, 50),
                    backgroundColor: AppTokens.cta,
                    foregroundColor: Colors.white,
                    shape: const StadiumBorder(),
                  ),
                  child: Text(actionLabel!),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
