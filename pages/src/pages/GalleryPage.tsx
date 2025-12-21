import GalleryGrid from "@/components/gallery-grid";

export default function GalleryPage() {
  return (
    <section className="py2 md:py-10 pb-4 md:pb-1">
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-semibold text-default-900 dark:text-default-700">Gallery</h1>
      </div>
      <GalleryGrid />
    </section>
  );
}
