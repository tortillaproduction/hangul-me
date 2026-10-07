const APP_NAME = 'Hangul-me';

interface NeonLogoProps {
  size?: number;
  /** アクセントのドットを明滅させる（ログインページ用） */
  flicker?: boolean;
  className?: string;
}

/**
 * ネオン管をイメージしたアプリロゴ。
 * ぼかした輪郭(glow-layer)のうえに発行する文字(core-layer)を重ねています。
 */
export function NeonLogo({ size = 18, flicker = false, className = '' }: NeonLogoProps) {
  return (
    <span
      className={`neon-logo ${className}`}
      style={{ fontSize: size, lineHeight: 1 }}
      aria-label={APP_NAME}
      role="img"
    >
      <span className="glow-layer" aria-hidden="true">
        {APP_NAME}
      </span>
      <span className="core-layer">{APP_NAME}</span>
      <span
        className={`flicker-dot ${flicker ? 'animate-flicker' : ''}`}
        aria-hidden="true"
        style={{
          width: Math.max(3, size * 0.11),
          height: Math.max(3, size * 0.11),
          right: -Math.max(4, size * 0.2),
          top: size * 0.12,
        }}
      />
    </span>
  );
}
