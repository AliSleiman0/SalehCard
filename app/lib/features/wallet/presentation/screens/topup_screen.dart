import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../domain/entities/wallet.dart';
import '../providers.dart';

/// Top-up screen: preset amount chips + a custom amount field, an out-of-band
/// payment channel selector (Whish/OMT/cash/USDT), an optional payment note,
/// and a request history list. Submits a PENDING request via
/// `POST /wallet/topups` — the wallet is credited only when an admin approves
/// after confirming the payment.
class TopUpScreen extends ConsumerStatefulWidget {
  const TopUpScreen({super.key});

  @override
  ConsumerState<TopUpScreen> createState() => _TopUpScreenState();
}

class _TopUpScreenState extends ConsumerState<TopUpScreen> {
  static const List<double> _presets = [25, 50, 100, 250];
  static const List<(String, IconData)> _channels = [
    ('whish', Icons.phone_iphone_rounded),
    ('omt', Icons.storefront_rounded),
    ('cash', Icons.payments_rounded),
    ('usdt', Icons.currency_exchange_rounded),
    ('other', Icons.more_horiz_rounded),
  ];

  final _customController = TextEditingController();
  final _noteController = TextEditingController();
  double? _selectedPreset = 50;
  String _channel = 'whish';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) ref.read(topUpControllerProvider.notifier).reset();
    });
  }

  @override
  void dispose() {
    _customController.dispose();
    _noteController.dispose();
    super.dispose();
  }

  /// The effective amount: a typed custom value takes precedence, else the chip.
  double get _amount {
    final custom = double.tryParse(_customController.text.trim());
    if (custom != null && custom > 0) return custom;
    return _selectedPreset ?? 0;
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context);
    final amount = _amount;
    if (amount <= 0) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.topUpAmount)),
      );
      return;
    }
    final req = await ref.read(topUpControllerProvider.notifier).submit(
          TopUpInput(
            amount: amount,
            channel: _channel,
            note: _noteController.text.trim(),
          ),
        );
    if (!mounted) return;
    if (req != null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(AppLocalizations.of(context).topUpRequested)),
      );
      _noteController.clear();
    } else {
      final failure = ref.read(topUpControllerProvider).failure;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(failure?.message.isNotEmpty == true
              ? failure!.message
              : l10n.paymentFailed),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final submitting = ref.watch(topUpControllerProvider).submitting;
    final requestsAsync = ref.watch(topUpRequestsProvider);
    final amount = _amount;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.topUpCta),
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(20, 18, 20, 24),
              children: [
                Text(
                  l10n.topUpAmount,
                  style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w800,
                      color: colors.text),
                ),
                const SizedBox(height: 12),
                Wrap(
                  spacing: 12,
                  runSpacing: 12,
                  children: [
                    for (final preset in _presets)
                      _AmountChip(
                        label: formatUsd(preset),
                        selected: _selectedPreset == preset &&
                            _customController.text.trim().isEmpty,
                        onTap: () {
                          FocusScope.of(context).unfocus();
                          setState(() {
                            _selectedPreset = preset;
                            _customController.clear();
                          });
                        },
                      ),
                  ],
                ),
                const SizedBox(height: 16),
                _CustomAmountField(
                  controller: _customController,
                  onChanged: (_) => setState(() {}),
                ),
                const SizedBox(height: 24),
                Text(
                  l10n.topUpVia,
                  style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w800,
                      color: colors.text),
                ),
                const SizedBox(height: 12),
                Wrap(
                  spacing: 10,
                  runSpacing: 10,
                  children: [
                    for (final (channel, icon) in _channels)
                      _ChannelChip(
                        label: channel.toUpperCase(),
                        icon: icon,
                        selected: _channel == channel,
                        onTap: () => setState(() => _channel = channel),
                      ),
                  ],
                ),
                const SizedBox(height: 16),
                _NoteField(controller: _noteController),
                const SizedBox(height: 26),
                Text(
                  l10n.topUpRequestsTitle,
                  style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w800,
                      color: colors.text),
                ),
                const SizedBox(height: 10),
                requestsAsync.when(
                  loading: () => const Padding(
                    padding: EdgeInsets.symmetric(vertical: 20),
                    child: Center(child: AppSpinner(size: 22)),
                  ),
                  error: (_, stack) => const SizedBox.shrink(),
                  data: (requests) => requests.isEmpty
                      ? const SizedBox.shrink()
                      : Column(
                          children: [
                            for (final req in requests)
                              _RequestTile(request: req, l10n: l10n),
                          ],
                        ),
                ),
              ],
            ),
          ),
          _TopUpBar(
            label: '${l10n.topUpCta}  ·  ${formatUsd(amount)}',
            submitting: submitting,
            enabled: amount > 0,
            colors: colors,
            onTap: _submit,
          ),
        ],
      ),
    );
  }
}

/// A selectable out-of-band payment channel chip.
class _ChannelChip extends StatelessWidget {
  const _ChannelChip({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 11),
        decoration: BoxDecoration(
          gradient: selected ? AppTokens.brandGradient : null,
          color: selected ? null : colors.surface,
          border: selected ? null : Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon,
                size: 16, color: selected ? Colors.white : colors.textDim),
            const SizedBox(width: 7),
            Text(
              label,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w800,
                color: selected ? Colors.white : colors.text,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Optional payment-reference note.
class _NoteField extends StatelessWidget {
  const _NoteField({required this.controller});

  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(color: colors.border),
    );
    return TextField(
      controller: controller,
      maxLength: 500,
      maxLines: 2,
      minLines: 1,
      style: TextStyle(fontSize: 14, color: colors.text),
      cursorColor: AppTokens.brand1,
      decoration: InputDecoration(
        filled: true,
        fillColor: colors.surface,
        counterText: '',
        hintText: AppLocalizations.of(context).topUpNoteHint,
        hintStyle: TextStyle(color: colors.textFaint),
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        border: border,
        enabledBorder: border,
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          borderSide: const BorderSide(color: AppTokens.brand1, width: 1.6),
        ),
      ),
    );
  }
}

/// One row in the request history: amount + channel + status chip (+ rejection
/// reason when present).
class _RequestTile extends StatelessWidget {
  const _RequestTile({required this.request, required this.l10n});

  final TopUpRequest request;
  final AppLocalizations l10n;

  (String, Color) _statusView() {
    switch (request.status) {
      case TopUpStatus.approved:
        return (l10n.statusApproved, AppTokens.accent);
      case TopUpStatus.rejected:
        return (l10n.statusRejected, AppTokens.danger);
      case TopUpStatus.pending:
      case TopUpStatus.unknown:
        return (l10n.statusPending, AppTokens.brand1);
    }
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final (statusLabel, statusColor) = _statusView();
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Text(
                formatUsd(request.amount),
                style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w800,
                    color: colors.text),
              ),
              const SizedBox(width: 8),
              Text(
                request.channel.toUpperCase(),
                style: TextStyle(
                    fontSize: 11.5,
                    fontWeight: FontWeight.w700,
                    color: colors.textFaint),
              ),
              const Spacer(),
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 9, vertical: 4),
                decoration: BoxDecoration(
                  color: statusColor.withValues(alpha: 0.14),
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  statusLabel,
                  style: TextStyle(
                    fontSize: 11.5,
                    fontWeight: FontWeight.w800,
                    color: statusColor,
                  ),
                ),
              ),
            ],
          ),
          if (request.status == TopUpStatus.rejected &&
              request.decisionReason.isNotEmpty) ...[
            const SizedBox(height: 6),
            Text(
              request.decisionReason,
              style: const TextStyle(
                  fontSize: 12.5,
                  fontWeight: FontWeight.w600,
                  color: AppTokens.danger),
            ),
          ],
        ],
      ),
    );
  }
}

class _AmountChip extends StatelessWidget {
  const _AmountChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        width: 96,
        padding: const EdgeInsets.symmetric(vertical: 16),
        alignment: Alignment.center,
        decoration: BoxDecoration(
          gradient: selected ? AppTokens.brandGradient : null,
          color: selected ? null : colors.surface,
          border: selected ? null : Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontSize: 15,
            fontWeight: FontWeight.w800,
            color: selected ? Colors.white : colors.text,
          ),
        ),
      ),
    );
  }
}

class _CustomAmountField extends StatelessWidget {
  const _CustomAmountField({required this.controller, required this.onChanged});

  final TextEditingController controller;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(color: colors.border),
    );
    return TextField(
      controller: controller,
      onChanged: onChanged,
      keyboardType: const TextInputType.numberWithOptions(decimal: true),
      inputFormatters: [
        FilteringTextInputFormatter.allow(RegExp(r'[0-9.]')),
      ],
      style: TextStyle(fontSize: 15, color: colors.text),
      cursorColor: AppTokens.brand1,
      decoration: InputDecoration(
        filled: true,
        fillColor: colors.surface,
        prefixIcon: Icon(Icons.attach_money_rounded, color: colors.textFaint),
        hintText: AppLocalizations.of(context).amountLabel,
        hintStyle: TextStyle(color: colors.textFaint),
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
        border: border,
        enabledBorder: border,
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          borderSide: const BorderSide(color: AppTokens.brand1, width: 1.6),
        ),
      ),
    );
  }
}

class _TopUpBar extends StatelessWidget {
  const _TopUpBar({
    required this.label,
    required this.submitting,
    required this.enabled,
    required this.colors,
    required this.onTap,
  });

  final String label;
  final bool submitting;
  final bool enabled;
  final AppColors colors;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.fromLTRB(
          18, 13, 18, 13 + MediaQuery.of(context).padding.bottom),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border(top: BorderSide(color: colors.border)),
      ),
      child: FilledButton(
        onPressed: (submitting || !enabled) ? null : onTap,
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(54),
          backgroundColor: AppTokens.cta,
          disabledBackgroundColor: AppTokens.cta.withValues(alpha: 0.5),
          foregroundColor: Colors.white,
          disabledForegroundColor: Colors.white,
          shape: const StadiumBorder(),
          elevation: 8,
          shadowColor: AppTokens.cta.withValues(alpha: 0.3),
          textStyle:
              const TextStyle(fontSize: 16.5, fontWeight: FontWeight.w800),
        ),
        child: submitting
            ? const AppSpinner(color: Colors.white, size: 22, stroke: 2.5)
            : Text(label),
      ),
    );
  }
}
