import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../data/order_field_upload.dart';

/// Checkout input for a `file` product field (a QR code / barcode photo): picks
/// an image, uploads it right away and writes the stored URL into
/// [controller] — so the parent reads it like any text field. While an upload
/// runs the value stays empty, which keeps the required-field check blocking.
class PhotoFieldInput extends ConsumerStatefulWidget {
  const PhotoFieldInput({
    super.key,
    required this.controller,
    this.onChanged,
    this.hasError = false,
  });

  final TextEditingController controller;
  final ValueChanged<String>? onChanged;
  final bool hasError;

  @override
  ConsumerState<PhotoFieldInput> createState() => _PhotoFieldInputState();
}

class _PhotoFieldInputState extends ConsumerState<PhotoFieldInput> {
  bool _uploading = false;

  void _set(String url) {
    setState(() => widget.controller.text = url);
    widget.onChanged?.call(url);
  }

  Future<void> _pick() async {
    FocusScope.of(context).unfocus();
    final l10n = AppLocalizations.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final picked = await ImagePicker()
        .pickImage(source: ImageSource.gallery, imageQuality: 85, maxWidth: 2000);
    if (picked == null || !mounted) return;
    setState(() => _uploading = true);
    try {
      final url = await ref.read(orderFieldUploaderProvider)(picked.path);
      if (!mounted) return;
      _set(url);
    } catch (_) {
      messenger.showSnackBar(SnackBar(content: Text(l10n.uploadFailed)));
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final url = widget.controller.text.trim();
    final borderColor = widget.hasError ? AppTokens.danger : colors.border;

    if (url.isNotEmpty) {
      return Row(
        children: [
          ClipRRect(
            borderRadius: BorderRadius.circular(AppTokens.rMd),
            child: Image.network(
              url,
              width: 72,
              height: 72,
              fit: BoxFit.cover,
              errorBuilder: (context, error, stack) => Container(
                width: 72,
                height: 72,
                color: colors.surface,
                child: const Icon(Icons.image_not_supported_outlined),
              ),
            ),
          ),
          const SizedBox(width: 12),
          TextButton.icon(
            onPressed: _uploading ? null : _pick,
            icon: const Icon(Icons.refresh, size: 18),
            label: Text(l10n.replacePhoto),
          ),
          IconButton(
            onPressed: _uploading ? null : () => _set(''),
            icon: const Icon(Icons.close),
          ),
        ],
      );
    }

    return OutlinedButton.icon(
      onPressed: _uploading ? null : _pick,
      style: OutlinedButton.styleFrom(
        minimumSize: const Size.fromHeight(52),
        side: BorderSide(color: borderColor),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
      ),
      icon: _uploading
          ? const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.qr_code_scanner),
      label: Text(l10n.uploadPhoto),
    );
  }
}
