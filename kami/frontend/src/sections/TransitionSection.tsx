interface Props {
  line: string
}

export function TransitionSection({ line }: Props) {
  return (
    <section
      id="transition"
      data-kami="section:transition"
      className="section flex items-center justify-center bg-rh-red-50 text-white px-8"
      aria-label="Transition"
    >
      <div className="text-center">
        <p className="text-5xl font-black tracking-tight leading-tight max-w-3xl">
          {line}
        </p>
      </div>
    </section>
  )
}
