interface BreadcrumbItem {
  id: string | null;
  label: string;
}

interface BreadcrumbsProps {
  items: BreadcrumbItem[];
  onNavigate: (id: string | null) => void;
}

export function Breadcrumbs({ items, onNavigate }: BreadcrumbsProps) {
  return (
    <nav className="flex items-center gap-1 text-xs text-gray-400 px-3 py-1.5 bg-gray-900/50 border-b border-gray-800">
      {items.map((item, i) => (
        <span key={item.id ?? "root"} className="flex items-center gap-1">
          {i > 0 && <span className="text-gray-600">›</span>}
          <button
            onClick={() => onNavigate(item.id)}
            className={`hover:text-white transition-colors ${
              i === items.length - 1 ? "text-gray-200 font-medium" : ""
            }`}
          >
            {item.label}
          </button>
        </span>
      ))}
    </nav>
  );
}
