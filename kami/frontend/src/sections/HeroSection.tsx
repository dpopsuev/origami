import type { HeroData } from '../hooks/useKabuki'

interface Props {
  data: HeroData
}

export function HeroSection({ data }: Props) {
  return (
    <section
      id="hero"
      data-kami="section:hero"
      className="section flex flex-col items-center justify-center bg-accent-surface text-fg-on-accent px-8"
      aria-label="Introduction"
    >
      <div className="text-center max-w-3xl">
        <div className="text-6xl font-black tracking-tight mb-4">
          <span className="text-brand">{data.title}</span>
        </div>
        <p className="text-2xl font-light text-fg-on-accent/80 mb-8">{data.subtitle}</p>
        {data.framework && (
          <p className="text-el-air text-lg mb-2">
            Powered by <span className="font-semibold text-el-earth">{data.framework}</span>
          </p>
        )}
        {data.presenter && (
          <p className="text-fg-faint text-sm mt-4">{data.presenter}</p>
        )}
        <div className="mt-12 text-fg-faint text-sm animate-bounce">
          ↓ Scroll to explore
        </div>
      </div>
    </section>
  )
}
