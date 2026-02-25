import type { HeroData } from '../hooks/usePresentation'

interface Props {
  data: HeroData
}

export function HeroSection({ data }: Props) {
  return (
    <section
      id="hero"
      data-kami="section:hero"
      className="section flex flex-col items-center justify-center bg-rh-gray-80 text-white px-8"
      aria-label="Introduction"
    >
      <div className="text-center max-w-3xl">
        <div className="text-6xl font-black tracking-tight mb-4">
          <span className="text-rh-red-50">{data.title}</span>
        </div>
        <p className="text-2xl font-light text-rh-gray-20 mb-8">{data.subtitle}</p>
        {data.framework && (
          <p className="text-rh-teal-30 text-lg mb-2">
            Powered by <span className="font-semibold text-rh-purple-50">{data.framework}</span>
          </p>
        )}
        {data.presenter && (
          <p className="text-rh-gray-40 text-sm mt-4">{data.presenter}</p>
        )}
        <div className="mt-12 text-rh-gray-40 text-sm animate-bounce">
          ↓ Scroll to explore
        </div>
      </div>
    </section>
  )
}
