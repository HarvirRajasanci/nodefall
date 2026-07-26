export default function AuthLayout({ children }) {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center gap-6">
      <div className="relative w-14 h-14 flex items-center justify-center">
        <div className="absolute inset-0 border-[1.5px] border-dashed border-emerald-500/60 rounded-full animate-[spin_8s_linear_infinite]" />
        <div className="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center text-gray-950 font-bold text-sm font-mono">
          N
        </div>
      </div>

      <div className="text-center">
        <div className="text-gray-100 text-lg font-medium font-mono tracking-[0.15em]">
          NODEFALL
        </div>
        <div className="text-gray-600 text-[11px] tracking-[0.08em] mt-0.5">
          LAST NODE STANDING
        </div>
      </div>

      <div className="relative bg-gray-900 border border-gray-800 rounded-xl p-8 w-full max-w-sm">
        <div className="absolute -top-px -left-px w-4 h-4 border-t-2 border-l-2 border-emerald-500 rounded-tl-xl" />
        <div className="absolute -bottom-px -right-px w-4 h-4 border-b-2 border-r-2 border-emerald-500 rounded-br-xl" />
        {children}
      </div>
    </div>
  );
}
