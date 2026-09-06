//
//  ZoomableImageView.swift
//  wee
//
//  Zoomable image viewer using UIScrollView
//

import SwiftUI

struct ZoomableImageView: UIViewRepresentable {
    let image: UIImage

    func makeUIView(context: Context) -> UIScrollView {
        let scrollView = UIScrollView()
        scrollView.delegate = context.coordinator
        scrollView.maximumZoomScale = 3.0
        scrollView.minimumZoomScale = 1.0

        let imageView = UIImageView(image: image)
        imageView.contentMode = .scaleAspectFit
        scrollView.addSubview(imageView)

        // Set up constraints to fit initial size
        imageView.frame = scrollView.bounds
        imageView.autoresizingMask = [.flexibleWidth, .flexibleHeight]

        context.coordinator.imageView = imageView

        return scrollView
    }

    func updateUIView(_ uiView: UIScrollView, context: Context) {
        // Update image if it changes
        if let imageView = context.coordinator.imageView {
            imageView.image = image
            // Recalculate zoom to fit
            uiView.setZoomScale(1.0, animated: false)
        }
    }

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    class Coordinator: NSObject, UIScrollViewDelegate {
        var imageView: UIImageView?

        func viewForZooming(in scrollView: UIScrollView) -> UIView? {
            return imageView
        }

        func scrollViewDidZoom(_ scrollView: UIScrollView) {
            // Center the image when zooming
            let offsetX = max(0, (scrollView.bounds.width - scrollView.contentSize.width) * 0.5)
            let offsetY = max(0, (scrollView.bounds.height - scrollView.contentSize.height) * 0.5)
            scrollView.contentInset = UIEdgeInsets(top: offsetY, left: offsetX, bottom: 0, right: 0)
        }
    }
}
