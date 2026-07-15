import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/network/idempotency.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../payments/presentation/providers.dart';
import '../../domain/entities/wallet.dart';
import '../providers.dart';

/// Sentinel selection key for the on-chain USDT option (settles automatically,
/// distinct from the admin-defined manual methods).
const _usdtOnchainKey = '__usdt_onchain__';

/// Top-up screen: preset amount chips + a custom amount field, a payment-method
/// selector (on-chain USDT when enabled + admin-defined manual methods, each
/// with its own instructions and inputs), an optional note, and a request
/// history list. A manual method submits a PENDING request via
/// `POST /wallet/topups` — the wallet is credited only when an admin approves
/// after confirming the payment. USDT settles on-chain automatically.
class TopUpScreen extends ConsumerStatefulWidget {
  const TopUpScreen({super.key});

  @override
  ConsumerState<TopUpScreen> createState() => _TopUpScreenState();
}

class _TopUpScreenState extends ConsumerState<TopUpScreen> {
  static const List<double> _presets = [25, 50, 100, 250];

  final _customController = TextEditingController();
  final _noteController = TextEditingController();
  double? _selectedPreset = 50;

  /// The selected method key: a method id, the on-chain USDT sentinel, or null
  /// (defaults to the first available option at render time).
  String? _selKey;
  // On-chain USDT network pick; empty = the server's default. Only offered
  // when the config lists more than one network.
  String _usdtNetwork = '';

  /// Collected values for the selected manual method's inputs (field key ->
  /// value; file fields hold the uploaded document URL).
  final Map<String, String> _fieldValues = {};

  /// File-field keys currently uploading (spinner state).
  final Set<String> _uploadingKeys = {};

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

  /// Selecting a method clears any inputs from the previously-selected one.
  void _selectMethod(String key) {
    if (_selKey == key) return;
    setState(() {
      _selKey = key;
      _fieldValues.clear();
      _uploadingKeys.clear();
    });
  }

  /// Picks a photo and uploads it, storing the returned URL as [fieldKey]'s
  /// value. Reuses the same image_picker → multipart flow as KYC.
  Future<void> _pickAndUpload(String fieldKey) async {
    FocusScope.of(context).unfocus();
    final l10n = AppLocalizations.of(context);
    final picked = await ImagePicker()
        .pickImage(source: ImageSource.gallery, imageQuality: 85, maxWidth: 2000);
    if (picked == null || !mounted) return;
    setState(() => _uploadingKeys.add(fieldKey));
    final result =
        await ref.read(walletRepositoryProvider).uploadDocument(picked.path);
    if (!mounted) return;
    result.match(
      (failure) {
        setState(() => _uploadingKeys.remove(fieldKey));
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text(failure.message.isNotEmpty
              ? failure.message
              : l10n.paymentFailed),
        ));
      },
      (url) {
        setState(() {
          _fieldValues[fieldKey] = url;
          _uploadingKeys.remove(fieldKey);
        });
      },
    );
  }

  Future<void> _submit(List<TopUpMethod> methods) async {
    final l10n = AppLocalizations.of(context);
    final amount = _amount;
    if (amount <= 0) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(l10n.topUpAmount)));
      return;
    }

    final selKey = _effectiveSelKey(methods);

    // On-chain USDT: create a deposit intent and hand off to the waiting screen.
    if (selKey == _usdtOnchainKey) {
      final config = await ref.read(paymentConfigProvider.future);
      if (!mounted) return;
      final network = config.networks.contains(_usdtNetwork) ? _usdtNetwork : '';
      await _submitUsdtIntent(amount, l10n, network);
      return;
    }

    final method = methods.where((m) => m.id == selKey).firstOrNull;
    if (method == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Please choose a payment method.')),
      );
      return;
    }

    // Required-field guard (the server re-validates).
    for (final f in method.fields) {
      if (f.isRequired && (_fieldValues[f.key]?.trim().isEmpty ?? true)) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${f.label} is required.')),
        );
        return;
      }
    }

    final fields = <String, String>{
      for (final e in _fieldValues.entries)
        if (e.value.trim().isNotEmpty) e.key: e.value.trim(),
    };

    final req = await ref.read(topUpControllerProvider.notifier).submit(
          TopUpInput(
            amount: amount,
            methodId: method.id,
            fields: fields,
            note: _noteController.text.trim(),
          ),
        );
    if (!mounted) return;
    if (req != null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.topUpRequested)),
      );
      _noteController.clear();
      setState(() {
        _fieldValues.clear();
        _uploadingKeys.clear();
      });
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

  /// Creates an on-chain USDT top-up intent and pushes the deposit screen.
  Future<void> _submitUsdtIntent(
    double amount,
    AppLocalizations l10n,
    String network,
  ) async {
    final intent = await ref
        .read(createTopUpIntentControllerProvider.notifier)
        .submit(amount, idempotencyKey: newIdempotencyKey(), network: network);
    if (!mounted) return;
    if (intent != null) {
      context.push('/payments/usdt-deposit', extra: intent);
    } else {
      final failure = ref.read(createTopUpIntentControllerProvider).failure;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(failure?.message.isNotEmpty == true
              ? failure!.message
              : l10n.paymentFailed),
        ),
      );
    }
  }

  /// Pull-to-refresh: an admin approval moves both the request status and the
  /// wallet balance, and the admin may have edited the method list.
  Future<void> _refreshRequests() async {
    ref.invalidate(topUpRequestsProvider);
    ref.invalidate(topUpMethodsProvider);
    ref.invalidate(walletProvider);
    await ref.read(topUpRequestsProvider.future);
  }

  /// The selection to act on: the explicit pick, else the first available
  /// option (on-chain USDT first when enabled, else the first manual method).
  String? _effectiveSelKey(List<TopUpMethod> methods) {
    if (_selKey != null) return _selKey;
    final usdtEnabled =
        ref.read(paymentConfigProvider).asData?.value.usdtEnabled ?? false;
    if (usdtEnabled) return _usdtOnchainKey;
    return methods.firstOrNull?.id;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final submitting = ref.watch(topUpControllerProvider).submitting ||
        ref.watch(createTopUpIntentControllerProvider).submitting;
    final requestsAsync = ref.watch(topUpRequestsProvider);
    final methodsAsync = ref.watch(topUpMethodsProvider);
    final amount = _amount;
    final paymentConfig = ref.watch(paymentConfigProvider).asData?.value;
    final usdtEnabled = paymentConfig?.usdtEnabled ?? false;
    final usdtNetworks = usdtEnabled ? paymentConfig!.networks : const <String>[];
    final methods = methodsAsync.asData?.value ?? const <TopUpMethod>[];
    final selKey = _effectiveSelKey(methods);
    final selectedMethod = methods.where((m) => m.id == selKey).firstOrNull;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.topUpCta),
      ),
      body: Column(
        children: [
          Expanded(
            child: RefreshIndicator(
              onRefresh: _refreshRequests,
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.fromLTRB(20, 18, 20, 24),
                children: [
                  Text(
                    l10n.topUpAmount,
                    style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w800,
                      color: colors.text,
                    ),
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
                      color: colors.text,
                    ),
                  ),
                  const SizedBox(height: 12),
                  if (methodsAsync.isLoading)
                    const Padding(
                      padding: EdgeInsets.symmetric(vertical: 8),
                      child: AppSpinner(size: 22),
                    ),
                  Wrap(
                    spacing: 10,
                    runSpacing: 10,
                    children: [
                      if (usdtEnabled)
                        _ChannelChip(
                          label: 'USDT',
                          icon: Icons.currency_exchange_rounded,
                          selected: selKey == _usdtOnchainKey,
                          onTap: () => _selectMethod(_usdtOnchainKey),
                        ),
                      for (final m in methods)
                        _ChannelChip(
                          label: m.name,
                          icon: Icons.account_balance_rounded,
                          selected: selKey == m.id,
                          onTap: () => _selectMethod(m.id),
                        ),
                    ],
                  ),
                  if (selKey == _usdtOnchainKey && usdtNetworks.length > 1) ...[
                    const SizedBox(height: 16),
                    Text(
                      l10n.usdtNetworkLabel,
                      style: TextStyle(
                        fontSize: 15,
                        fontWeight: FontWeight.w800,
                        color: colors.text,
                      ),
                    ),
                    const SizedBox(height: 12),
                    Wrap(
                      spacing: 10,
                      runSpacing: 10,
                      children: [
                        for (final network in usdtNetworks)
                          _ChannelChip(
                            label: network.toUpperCase(),
                            icon: Icons.hub_rounded,
                            selected: (_usdtNetwork.isEmpty
                                    ? usdtNetworks.first
                                    : _usdtNetwork) ==
                                network,
                            onTap: () =>
                                setState(() => _usdtNetwork = network),
                          ),
                      ],
                    ),
                  ],
                  if (selectedMethod != null) ...[
                    if (selectedMethod.instructions.isNotEmpty)
                      _InstructionsCard(text: selectedMethod.instructions),
                    for (final field in selectedMethod.fields)
                      Padding(
                        padding: const EdgeInsets.only(top: 14),
                        child: _MethodFieldInput(
                          field: field,
                          value: _fieldValues[field.key],
                          uploading: _uploadingKeys.contains(field.key),
                          onChanged: (v) =>
                              setState(() => _fieldValues[field.key] = v),
                          onPickFile: () => _pickAndUpload(field.key),
                        ),
                      ),
                  ],
                  const SizedBox(height: 16),
                  _NoteField(controller: _noteController),
                  const SizedBox(height: 26),
                  Text(
                    l10n.topUpRequestsTitle,
                    style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w800,
                      color: colors.text,
                    ),
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
          ),
          _TopUpBar(
            label: '${l10n.topUpCta}  ·  ${formatUsd(amount)}',
            submitting: submitting,
            enabled: amount > 0 && selKey != null,
            colors: colors,
            onTap: () => _submit(methods),
          ),
        ],
      ),
    );
  }
}

/// A card showing the selected method's "how to pay" instructions.
class _InstructionsCard extends StatelessWidget {
  const _InstructionsCard({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      margin: const EdgeInsets.only(top: 14),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: AppTokens.brand1.withValues(alpha: 0.08),
        border: Border.all(color: AppTokens.brand1.withValues(alpha: 0.3)),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.info_outline_rounded, size: 18, color: AppTokens.brand1),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              text,
              style: TextStyle(
                fontSize: 13.5,
                height: 1.4,
                color: colors.text,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// One dynamic input for a manual method: text/number → field, select →
/// dropdown, file → pick-and-upload button with an uploaded/loading state.
class _MethodFieldInput extends StatelessWidget {
  const _MethodFieldInput({
    required this.field,
    required this.value,
    required this.uploading,
    required this.onChanged,
    required this.onPickFile,
  });

  final TopUpMethodField field;
  final String? value;
  final bool uploading;
  final ValueChanged<String> onChanged;
  final VoidCallback onPickFile;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final label = field.isRequired ? '${field.label} *' : field.label;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: TextStyle(
            fontSize: 13.5,
            fontWeight: FontWeight.w700,
            color: colors.text,
          ),
        ),
        const SizedBox(height: 8),
        switch (field.type) {
          TopUpFieldType.select => _SelectField(
              field: field,
              value: value,
              onChanged: onChanged,
            ),
          TopUpFieldType.file => _FileField(
              uploaded: value != null && value!.isNotEmpty,
              uploading: uploading,
              onPick: onPickFile,
            ),
          _ => _TextInput(
              field: field,
              value: value,
              onChanged: onChanged,
            ),
        },
      ],
    );
  }
}

class _TextInput extends StatelessWidget {
  const _TextInput({
    required this.field,
    required this.value,
    required this.onChanged,
  });

  final TopUpMethodField field;
  final String? value;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final isNumber = field.type == TopUpFieldType.number;
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(color: colors.border),
    );
    return TextField(
      keyboardType: isNumber
          ? const TextInputType.numberWithOptions(decimal: true)
          : TextInputType.text,
      inputFormatters: isNumber
          ? [FilteringTextInputFormatter.allow(RegExp(r'[0-9.]'))]
          : null,
      onChanged: onChanged,
      style: TextStyle(fontSize: 14, color: colors.text),
      cursorColor: AppTokens.brand1,
      decoration: InputDecoration(
        filled: true,
        fillColor: colors.surface,
        hintText: field.placeholder.isNotEmpty ? field.placeholder : null,
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

class _SelectField extends StatelessWidget {
  const _SelectField({
    required this.field,
    required this.value,
    required this.onChanged,
  });

  final TopUpMethodField field;
  final String? value;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(color: colors.border),
    );
    return DropdownButtonFormField<String>(
      initialValue:
          (value != null && field.options.contains(value)) ? value : null,
      isExpanded: true,
      dropdownColor: colors.surface,
      style: TextStyle(fontSize: 14, color: colors.text),
      hint: Text(field.placeholder.isNotEmpty ? field.placeholder : 'Select…',
          style: TextStyle(color: colors.textFaint)),
      decoration: InputDecoration(
        filled: true,
        fillColor: colors.surface,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        border: border,
        enabledBorder: border,
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          borderSide: const BorderSide(color: AppTokens.brand1, width: 1.6),
        ),
      ),
      items: [
        for (final opt in field.options)
          DropdownMenuItem(value: opt, child: Text(opt)),
      ],
      onChanged: (v) {
        if (v != null) onChanged(v);
      },
    );
  }
}

class _FileField extends StatelessWidget {
  const _FileField({
    required this.uploaded,
    required this.uploading,
    required this.onPick,
  });

  final bool uploaded;
  final bool uploading;
  final VoidCallback onPick;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final done = uploaded && !uploading;
    return InkWell(
      onTap: uploading ? null : onPick,
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        decoration: BoxDecoration(
          color: colors.surface,
          border: Border.all(
            color: done ? AppTokens.accent : colors.border,
          ),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Row(
          children: [
            if (uploading)
              const AppSpinner(size: 18)
            else
              Icon(
                done ? Icons.check_circle_rounded : Icons.upload_file_rounded,
                size: 20,
                color: done ? AppTokens.accent : colors.textDim,
              ),
            const SizedBox(width: 10),
            Text(
              uploading
                  ? 'Uploading…'
                  : done
                      ? 'Uploaded · tap to replace'
                      : 'Upload photo',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: done ? AppTokens.accent : colors.text,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A selectable payment-method chip.
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
            Icon(
              icon,
              size: 16,
              color: selected ? Colors.white : colors.textDim,
            ),
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
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 14,
        ),
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

/// One row in the request history: amount + method + status chip (+ rejection
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
                  color: colors.text,
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  request.methodLabel.toUpperCase(),
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 11.5,
                    fontWeight: FontWeight.w700,
                    color: colors.textFaint,
                  ),
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 4),
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
                color: AppTokens.danger,
              ),
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
      inputFormatters: [FilteringTextInputFormatter.allow(RegExp(r'[0-9.]'))],
      style: TextStyle(fontSize: 15, color: colors.text),
      cursorColor: AppTokens.brand1,
      decoration: InputDecoration(
        filled: true,
        fillColor: colors.surface,
        prefixIcon: Icon(Icons.attach_money_rounded, color: colors.textFaint),
        hintText: AppLocalizations.of(context).amountLabel,
        hintStyle: TextStyle(color: colors.textFaint),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 16,
        ),
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
        18,
        13,
        18,
        13 + MediaQuery.of(context).padding.bottom,
      ),
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
          textStyle: const TextStyle(
            fontSize: 16.5,
            fontWeight: FontWeight.w800,
          ),
        ),
        child: submitting
            ? const AppSpinner(color: Colors.white, size: 22, stroke: 2.5)
            : Text(label),
      ),
    );
  }
}
